import { Navigate, useParams } from "react-router-dom";

/** Legacy redirect: /app/applications/:id → /app/jobs/:id */
export function LegacyApplicationDetailRedirect() {
  const { id } = useParams<{ id: string }>();
  return <Navigate to={`/app/jobs/${id}`} replace />;
}
