import { Link, Route, Routes } from 'react-router';
import { Layout } from './components/Layout';
import { AboutPage } from './pages/AboutPage';
import { HomePage } from './pages/HomePage';
import { TeamPage } from './pages/TeamPage';

/**
 * The site's pages. Every page shares the Layout's header and footer.
 *
 * React Router note: each <Route> pairs an address with the component to show.
 * Routes nested inside the Layout route are drawn where Layout puts its
 * <Outlet />, so the header and footer stay while the middle changes.
 * "index" is the address "/" itself; ":id" names a changing part of the
 * address that the page reads with useParams; "*" catches anything else.
 *
 * Moving between pages happens in the browser with no full page load. That
 * is why the Go server answers every unknown address with index.html
 * (backend/internal/web): React then reads the address and picks the page.
 */
export function App() {
  return (
    <Routes>
      <Route element={<Layout />}>
        <Route index element={<HomePage />} />
        <Route path="teams/:id" element={<TeamPage />} />
        <Route path="about" element={<AboutPage />} />
        <Route
          path="*"
          element={
            <p>
              That page does not exist. <Link to="/">Back to this week’s games</Link>
            </p>
          }
        />
      </Route>
    </Routes>
  );
}
