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
  return (
    <div className="cli-command">
      <div className="cli-command__body">{children}</div>
      <div className="cli-command__footer">
        {badges[context].map((label) => (
          <span key={label} className={`cli-command__badge ${badgeClass[label]}`}>
            {label}
          </span>
        ))}
      </div>
    </div>
  );
}
