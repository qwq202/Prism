import { Provider } from "react-redux";
import store from "./store/index.ts";
import AppProvider from "./components/app/AppProvider.tsx";
import { RouterProvider } from "react-router-dom";
import router from "./router.tsx";
import { Toaster } from "@/components/ui/sonner";
import Spinner from "@/spinner.tsx";
import ReloadPrompt from "@/components/ReloadService.tsx";
import PersonalizationSync from "@/components/app/PersonalizationSync.tsx";

function App() {
  return (
    <Provider store={store}>
      <AppProvider>
        <PersonalizationSync />
        <Toaster />
        <Spinner />
        <ReloadPrompt />
        <RouterProvider router={router} future={{ v7_startTransition: true }} />
      </AppProvider>
    </Provider>
  );
}

export default App;
