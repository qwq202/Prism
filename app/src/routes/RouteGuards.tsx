import { useEffect } from "react";
import { useSelector } from "react-redux";
import { useLocation, useNavigate } from "react-router-dom";
import { selectAdmin, selectAuthenticated, selectInit } from "@/store/auth.ts";
import { AdminShellSkeleton } from "@/components/admin/AdminSkeleton.tsx";

export function AuthRequired({ children }: { children: React.ReactNode }) {
  const init = useSelector(selectInit);
  const authenticated = useSelector(selectAuthenticated);
  const navigate = useNavigate();
  const location = useLocation();

  useEffect(() => {
    if (init && !authenticated) {
      navigate("/login", { state: { from: location.pathname } });
    }
  }, [init, authenticated, location.pathname, navigate]);

  return <>{children}</>;
}

export function AuthForbidden({ children }: { children: React.ReactNode }) {
  const init = useSelector(selectInit);
  const authenticated = useSelector(selectAuthenticated);
  const navigate = useNavigate();
  const location = useLocation();

  useEffect(() => {
    if (init && authenticated) {
      navigate("/", { state: { from: location.pathname } });
    }
  }, [init, authenticated, location.pathname, navigate]);

  return <>{children}</>;
}

export function AdminRequired({ children }: { children: React.ReactNode }) {
  const init = useSelector(selectInit);
  const admin = useSelector(selectAdmin);
  const navigate = useNavigate();
  const location = useLocation();

  useEffect(() => {
    if (init && !admin) {
      navigate("/", { state: { from: location.pathname } });
    }
  }, [init, admin, location.pathname, navigate]);

  if (!init) return <AdminShellSkeleton />;
  if (!admin) return null;

  return <>{children}</>;
}
