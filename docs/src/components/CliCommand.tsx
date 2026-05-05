import React from "react";

type Context = "server" | "client" | "both";

const badges: Record<Context, string[]> = {
  server: ["SERVER"],
  client: ["CLIENT"],
  both: ["SERVER", "CLIENT"],
};

const badgeClass: Record<string, string> = {
  SERVER: "cli-command__badge--server",
  CLIENT: "cli-command__badge--client",
};

export default function CliCommand({
  context,
  children,
}: {
  context: Context;
  children: React.ReactNode;
}) {
  const labels = badges[context];
  const description = `Runs on: ${labels.join(" and ").toLowerCase()}`;

  return (
    <div className="cli-command" role="group" aria-label={description}>
      <div className="cli-command__body">{children}</div>
      <div className="cli-command__footer" aria-hidden="true">
        {labels.map((label) => (
          <span key={label} className={`cli-command__badge ${badgeClass[label]}`}>
            {label}
          </span>
        ))}
      </div>
    </div>
  );
}
