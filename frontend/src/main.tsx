import { StrictMode, lazy, Suspense } from "react";
import { createRoot } from "react-dom/client";
import "./styles.css";
import { DocumentApp } from "./doc/DocumentApp";

const LibraryApp = lazy(() => import("./library/LibraryApp"));
const PrefsApp = lazy(() => import("./prefs/PrefsApp"));
const BatchApp = lazy(() => import("./batch/BatchApp"));

function App() {
  const path = location.pathname.replace(/\/+$/, "");
  switch (path) {
    case "/library":
      return <LibraryApp />;
    case "/prefs":
      return <PrefsApp />;
    case "/batch":
      return <BatchApp />;
  }
  return <DocumentApp />;
}

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <Suspense fallback={<div className="page" />}>
      <App />
    </Suspense>
  </StrictMode>,
);
