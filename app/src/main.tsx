import ReactDOM from "react-dom/client";
import App from "./App.tsx";
import "./conf/bootstrap.ts";
import "./i18n.ts";
import "./assets/main.less";
import "./assets/globals.less";

ReactDOM.createRoot(document.getElementById("root")!).render(<App />);
