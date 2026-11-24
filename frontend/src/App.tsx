import { Link, Route, Routes } from 'react-router';
import { Layout } from './components/Layout';
import { AboutPage } from './pages/AboutPage';
import { HomePage } from './pages/HomePage';
import { TeamPage } from './pages/TeamPage';

/** The site's pages. Every page shares the Layout's header and footer. */
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
