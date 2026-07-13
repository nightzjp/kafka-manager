import React from 'react';
import ReactDOM from 'react-dom/client';
import { App } from './App';
import { ThemeProvider } from './theme/ThemeProvider';
import { AppErrorBoundary } from './components/AppErrorBoundary';
import { FeedbackProvider } from './components/Feedback';

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <AppErrorBoundary>
      <ThemeProvider><FeedbackProvider><App /></FeedbackProvider></ThemeProvider>
    </AppErrorBoundary>
  </React.StrictMode>,
);
