
  import { createRoot } from "react-dom/client";
  import { ThemeProvider } from "next-themes";
  import App from "./app/App.tsx";
  import "./styles/index.css";

  createRoot(document.getElementById("root")!).render(
    <ThemeProvider attribute="class" storageKey="mcpbox-theme" defaultTheme="system" enableSystem disableTransitionOnChange>
      <App />
    </ThemeProvider>,
  );
  