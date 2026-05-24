import { useEffect } from "react";
import { useSelector } from "react-redux";
import { useLocation, useNavigate } from "react-router-dom";
import { selectAdmin, selectAuthenticated, selectInit } from "@/store/auth.ts";
import { AdminShellSkeleton } from "@/components/admin/AdminSkeleton.tsx";
import Loader from "@/components/Loader.tsx";
import { useTranslation } from "react-i18next";

export function AuthRequired({ children }: { children: React.ReactNode }) {
  const { t } = useTranslation();
  const init = useSelector(selectInit);
  const authenticated = useSelector(selectAuthenticated);
  const navigate = useNavigate();
  const location = useLocation();
  const currentPath = `${location.pathname}${location.search}`;

  useEffect(() => {
    if (init && !authenticated) {
      navigate("/login", { replace: true, state: { from: currentPath } });
    }
  }, [init, authenticated, currentPath, navigate]);

  if (!init) return <Loader prompt={t("loading")} />;
  if (!authenticated) return null;

  return <>{children}</>;
}

export function AuthForbidden({ children }: { children: React.ReactNode }) {
  const { t } = useTranslation();
  const init = useSelector(selectInit);
  const authenticated = useSelector(selectAuthenticated);
  const navigate = useNavigate();

  useEffect(() => {
    if (init && authenticated) {
      navigate("/", { replace: true });
    }
  }, [init, authenticated, navigate]);

  if (!init) return <Loader prompt={t("loading")} />;
  if (authenticated) return null;

  return <>{children}</>;
}

export function AdminRequired({ children }: { children: React.ReactNode }) {
  const init = useSelector(selectInit);
  const authenticated = useSelector(selectAuthenticated);
  const admin = useSelector(selectAdmin);
  const navigate = useNavigate();
  const location = useLocation();
  const currentPath = `${location.pathname}${location.search}`;

  useEffect(() => {
    if (!init) return;

    if (!authenticated) {
      navigate("/login", { replace: true, state: { from: currentPath } });
      return;
    }

    if (!admin) {
      navigate("/", { replace: true });
    }
  }, [init, authenticated, admin, currentPath, navigate]);

  if (!init) return <AdminShellSkeleton />;
  if (!authenticated || !admin) return null;

  return <>{children}</>;
}
