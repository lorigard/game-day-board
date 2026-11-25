import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import { BrowserRouter } from 'react-router';
import { App } from './App';
import './styles.css';

// The starting point of the React app. index.html loads this file, and this
// file draws <App /> into the empty <div id="root"> in index.html.

const root = document.getElementById('root');
if (!root) throw new Error('index.html is missing the #root element');

// StrictMode adds extra checks while developing (for example, it runs effects
// twice to expose missing cleanup code); it does nothing in the built site.
// BrowserRouter keeps the address bar and the page on screen in step.
createRoot(root).render(
  <StrictMode>
    <BrowserRouter>
      <App />
    </BrowserRouter>
  </StrictMode>,
);
