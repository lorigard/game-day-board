package store

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/lorigard/game-day-board/backend/internal/football"
)

// Collection and document names. Layout (SRS 7.1):
//
//	games/{gameId}
//	teams/{teamId}
//	meta/calendar   {weeks: [...]}
//	meta/ingest     the last ingest run's status
const (
	gamesCollection = "games"
	teamsCollection = "teams"
	metaCollection  = "meta"
	calendarDoc     = "calendar"
	ingestDoc       = "ingest"
)

// Firestore is the production Store. When the FIRESTORE_EMULATOR_HOST
// environment variable is set, the client library talks to the local
// emulator instead, which is how the integration tests run.
type Firestore struct {
	client *firestore.Client
}

var _ Store = (*Firestore)(nil)

// NewFirestore connects to the default database of projectID.
func NewFirestore(ctx context.Context, projectID string) (*Firestore, error) {
	client, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("connecting to Firestore: %w", err)
	}
	return &Firestore{client: client}, nil
}

// Close releases the client's connections.
func (f *Firestore) Close() error { return f.client.Close() }

// calendarDocument wraps the weeks because a Firestore document must be a map, not a list.
type calendarDocument struct {
	Weeks []football.Week `firestore:"weeks"`
}

func (f *Firestore) SaveGames(ctx context.Context, games []football.Game) error {
	return f.bulkSet(ctx, len(games), func(i int) (*firestore.DocumentRef, any) {
		g := games[i].WithKeys()
		return f.client.Collection(gamesCollection).Doc(strconv.Itoa(g.ID)), g
	})
}

func (f *Firestore) SaveTeams(ctx context.Context, teams []football.Team) error {
	return f.bulkSet(ctx, len(teams), func(i int) (*firestore.DocumentRef, any) {
		return f.client.Collection(teamsCollection).Doc(strconv.Itoa(teams[i].ID)), teams[i]
	})
}

// bulkSet writes n documents in parallel batches. Set replaces the whole
// document, which is what makes a repeated ingest run harmless.
func (f *Firestore) bulkSet(ctx context.Context, n int, doc func(i int) (*firestore.DocumentRef, any)) error {
	if n == 0 {
		return nil
	}
	bw := f.client.BulkWriter(ctx)
	jobs := make([]*firestore.BulkWriterJob, 0, n)
	for i := 0; i < n; i++ {
		ref, data := doc(i)
		job, err := bw.Set(ref, data)
		if err != nil {
			bw.End()
			return fmt.Errorf("queueing write for %s: %w", ref.Path, err)
		}
		jobs = append(jobs, job)
	}
	bw.End() // waits for every queued write to finish
	var errs []error
	for _, job := range jobs {
		if _, err := job.Results(); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("%d of %d writes failed, first: %w", len(errs), n, errs[0])
	}
	return nil
}

func (f *Firestore) SaveCalendar(ctx context.Context, weeks []football.Week) error {
	_, err := f.client.Collection(metaCollection).Doc(calendarDoc).Set(ctx, calendarDocument{Weeks: weeks})
	return err
}

func (f *Firestore) SaveIngestStatus(ctx context.Context, s football.IngestStatus) error {
	_, err := f.client.Collection(metaCollection).Doc(ingestDoc).Set(ctx, s)
	return err
}

func (f *Firestore) GamesForWeek(ctx context.Context, season, week int, seasonType string) ([]football.Game, error) {
	// Equality filters only, so Firestore's automatic single-field indexes are
	// enough. Sorting happens in Go to avoid needing a composite index.
	q := f.client.Collection(gamesCollection).
		Where("season", "==", season).
		Where("week", "==", week).
		Where("seasonType", "==", seasonType)
	return readAll[football.Game](ctx, q.Documents(ctx))
}

func (f *Firestore) GamesForTeam(ctx context.Context, season, teamID int) ([]football.Game, error) {
	q := f.client.Collection(gamesCollection).
		Where("teamSeasonKeys", "array-contains", football.TeamSeasonKey(season, teamID))
	return readAll[football.Game](ctx, q.Documents(ctx))
}

func (f *Firestore) Team(ctx context.Context, id int) (football.Team, error) {
	var t football.Team
	err := f.get(ctx, f.client.Collection(teamsCollection).Doc(strconv.Itoa(id)), &t)
	return t, err
}

func (f *Firestore) Teams(ctx context.Context) ([]football.Team, error) {
	return readAll[football.Team](ctx, f.client.Collection(teamsCollection).Documents(ctx))
}

func (f *Firestore) Calendar(ctx context.Context) ([]football.Week, error) {
	var c calendarDocument
	err := f.get(ctx, f.client.Collection(metaCollection).Doc(calendarDoc), &c)
	if errors.Is(err, ErrNotFound) {
		return nil, nil // no calendar yet: an empty season, not an error
	}
	return c.Weeks, err
}

func (f *Firestore) IngestStatus(ctx context.Context) (football.IngestStatus, error) {
	var s football.IngestStatus
	err := f.get(ctx, f.client.Collection(metaCollection).Doc(ingestDoc), &s)
	if errors.Is(err, ErrNotFound) {
		return football.IngestStatus{CallsRemaining: -1}, nil
	}
	return s, err
}

// Ping reads a single small document. A missing document still proves the
// database answered; only a connection or permission failure is an error.
func (f *Firestore) Ping(ctx context.Context) error {
	_, err := f.client.Collection(metaCollection).Doc(ingestDoc).Get(ctx)
	if err != nil && status.Code(err) != codes.NotFound {
		return err
	}
	return nil
}

// get loads one document into dst, translating "no such document" to ErrNotFound.
func (f *Firestore) get(ctx context.Context, ref *firestore.DocumentRef, dst any) error {
	snap, err := ref.Get(ctx)
	if status.Code(err) == codes.NotFound {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	return snap.DataTo(dst)
}

// readAll decodes every document from an iterator into a slice of T.
func readAll[T any](ctx context.Context, it *firestore.DocumentIterator) ([]T, error) {
	defer it.Stop()
	var out []T
	for {
		snap, err := it.Next()
		if errors.Is(err, iterator.Done) {
			return out, nil
		}
		if err != nil {
			return nil, err
		}
		var v T
		if err := snap.DataTo(&v); err != nil {
			return nil, fmt.Errorf("decoding %s: %w", snap.Ref.Path, err)
		}
		out = append(out, v)
	}
}
