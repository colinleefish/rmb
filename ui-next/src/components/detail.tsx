import { Badge } from "@/components/ui/badge";

export function DetailBadges({ children }: { children: React.ReactNode }) {
  return <div className="flex flex-wrap items-center gap-2">{children}</div>;
}

export function OutlineBadge({ children }: { children: React.ReactNode }) {
  return (
    <Badge variant="outline" className="font-normal">
      {children}
    </Badge>
  );
}

export function DetailText({ children }: { children: React.ReactNode }) {
  return (
    <p className="text-foreground/90 text-sm whitespace-pre-wrap">{children}</p>
  );
}

export function DetailLead({ children }: { children: React.ReactNode }) {
  return (
    <p className="text-foreground text-sm font-medium whitespace-pre-wrap">
      {children}
    </p>
  );
}

export function DetailMeta({ children }: { children: React.ReactNode }) {
  return <p className="text-muted-foreground text-xs">{children}</p>;
}

export function DetailUri({ children }: { children: React.ReactNode }) {
  return (
    <p className="text-muted-foreground font-mono text-xs break-all">
      {children}
    </p>
  );
}
