import { NavLink, Outlet } from 'react-router';

export const REPO_URL = 'https://github.com/lorigard/game-day-board';

/** The frame around every page: header, navigation, and the attribution footer (FR-10). */
export function Layout() {
  return (
    <>
      <a className="skip-link" href="#main">
        Skip to content
      </a>
      <header className="site-header">
        <div className="container header-row">
          <NavLink to="/" className="site-title">
            Game Day Board
          </NavLink>
          <nav aria-label="Main">
            <NavLink to="/" end>
              Games
            </NavLink>
            <NavLink to="/about">About</NavLink>
          </nav>
        </div>
      </header>
      <main id="main" className="container">
        <Outlet />
      </main>
      <footer className="site-footer">
        <div className="container">
          <p>
            Data provided by <a href="https://collegefootballdata.com">CollegeFootballData.com</a>.
          </p>
          <p>
            A learning project by <a href="https://lori-gardner.com">Lori Gardner</a> · <a href={REPO_URL}>Source code</a>
          </p>
        </div>
      </footer>
    </>
  );
}
