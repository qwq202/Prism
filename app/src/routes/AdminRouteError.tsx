import "@/assets/common/404.less";
import { isRouteErrorResponse, useRouteError } from "react-router-dom";
import { Button } from "@/components/ui/button.tsx";
import { AlertTriangle } from "lucide-react";
import { useTranslation } from "react-i18next";
import NotFound from "@/routes/NotFound.tsx";

function getErrorMessage(error: unknown): string {
  if (isRouteErrorResponse(error)) {
    if (typeof error.data === "string" && error.data.trim().length > 0) {
      return error.data;
    }
    return error.statusText || `${error.status}`;
  }

  if (error instanceof Error) return error.message;
  if (typeof error === "string") return error;

  return "";
}

function AdminRouteError() {
  const { t } = useTranslation();
  const error = useRouteError();

  if (isRouteErrorResponse(error) && error.status === 404) {
    return <NotFound />;
  }

  return (
    <div className="error-page">
      <AlertTriangle className="icon" />
      <h1>{t("admin.error")}</h1>
      <p>{getErrorMessage(error) || t("request-failed")}</p>
      <Button onClick={() => window.location.reload()}>{t("try-again")}</Button>
    </div>
  );
}

export default AdminRouteError;
