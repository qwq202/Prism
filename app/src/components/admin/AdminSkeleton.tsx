import "@/assets/admin/all.less";
import { Skeleton } from "@/components/ui/skeleton.tsx";

function MenuSkeleton() {
  return (
    <div className="admin-menu" aria-hidden="true">
      {Array.from({ length: 9 }).map((_, index) => (
        <div className="menu-item" key={index}>
          <Skeleton className="h-7 w-7 rounded-md" />
        </div>
      ))}
    </div>
  );
}

export function AdminContentSkeleton() {
  return (
    <div className="admin-container" aria-hidden="true">
      <div className="admin-card rounded-lg bg-background p-6">
        <div className="mb-8 flex items-center justify-between gap-4">
          <div className="space-y-3">
            <Skeleton className="h-8 w-36" />
            <Skeleton className="h-4 w-56" />
          </div>
          <div className="flex gap-3">
            <Skeleton className="h-10 w-24" />
            <Skeleton className="h-10 w-20" />
          </div>
        </div>

        <div className="mb-8 grid gap-4 md:grid-cols-3">
          {Array.from({ length: 3 }).map((_, index) => (
            <div
              className="rounded-lg border border-border bg-card p-5"
              key={index}
            >
              <Skeleton className="mb-5 h-4 w-24" />
              <Skeleton className="mb-3 h-8 w-28" />
              <Skeleton className="h-3 w-full" />
            </div>
          ))}
        </div>

        <div className="rounded-lg border border-border">
          <div className="grid grid-cols-4 gap-4 border-b border-border p-4">
            {Array.from({ length: 4 }).map((_, index) => (
              <Skeleton className="h-4 w-20" key={index} />
            ))}
          </div>
          {Array.from({ length: 6 }).map((_, row) => (
            <div className="grid grid-cols-4 gap-4 p-4" key={row}>
              {Array.from({ length: 4 }).map((_, col) => (
                <Skeleton className="h-5 w-full" key={col} />
              ))}
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

export function AdminShellSkeleton() {
  return (
    <div className="home-page flex flex-row flex-1">
      <div className="admin-page">
        <MenuSkeleton />
        <div className="flex h-full flex-1 overflow-hidden bg-[hsla(var(--background-container))]">
          <AdminContentSkeleton />
        </div>
      </div>
    </div>
  );
}
