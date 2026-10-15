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
      document.title = "我的库";
      return <LibraryApp />;
    case "/prefs":
      document.title = "偏好设置";
      return <PrefsApp />;
    case "/batch":
      document.title = "批量处理文件";
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
