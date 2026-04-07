import type { Prism } from "prism-react-renderer";

export function registerMirenLanguage(PrismInstance: typeof Prism) {
  PrismInstance.languages["miren"] = {
    comment: {
      pattern: /#.*/,
      greedy: true,
    },
    string: [
      {
        pattern: /(["'])(?:\\[\s\S]|(?!\1)[^\\])*\1/,
        greedy: true,
      },
    ],
    variable: /\$\w+/,
    // Hint text in parentheses: (Use ↑/↓ or j/k ...)
    hint: {
      pattern: /^\(.*\)$/m,
      alias: "comment",
    },
    // Success checkmark lines: ✓ Something completed
    "status-success": {
      pattern: /✓.*/,
      alias: "inserted",
    },
    // Spinner/progress indicator: ⠋ ⠙ ⠹ etc or ::
    "status-progress": {
      pattern: /[⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏].*/,
      alias: "comment",
    },
    // Progress bar: █▓░ blocks or similar
    "progress-bar": {
      pattern: /[░▒▓█]{2,}/,
      alias: "builtin",
    },
    // Deploy output: "Your app is available at:"
    "deploy-url-label": {
      pattern: /Your app is available at:/,
      alias: "builtin",
    },
    // Version strings like demo-vCZpcrAAaU6mULzMSSBwc4
    "version-id": {
      pattern: /\b\w+-v[A-Za-z0-9]{10,}\b/,
      alias: "constant",
    },
    // Arrow in "demo → club"
    arrow: {
      pattern: /→/,
      alias: "operator",
    },
    // Table column headers: individual all-caps words (2+ chars)
    "table-header": {
      pattern: /\b[A-Z]{2,}\b/,
      alias: "tag",
    },
    // Selection indicator
    selector: {
      pattern: /▸/,
      alias: "keyword",
    },
    // Full miren command pattern: [sudo] miren <command> [<subcommand>]
    "miren-full": {
      pattern: /\b(sudo\s+)?(?:miren|m)\s+(\w[\w-]*)(?:\s+(\w[\w-]*))?/,
      inside: {
        sudo: {
          pattern: /^sudo\b/,
          alias: "keyword",
        },
        "miren-bin": {
          pattern: /\b(?:miren|m)\b/,
          alias: "function",
        },
        subcommand: {
          pattern: /\S+$/,
          alias: "property",
        },
        command: {
          pattern: /\S+/,
          alias: "builtin",
        },
      },
    },
    // Bare sudo (e.g. sudo systemctl ...)
    sudo: {
      pattern: /\bsudo\b/,
      alias: "keyword",
    },
    // IP address with optional :port
    ip: {
      pattern: /\d{1,3}(?:\.\d{1,3}){3}(?::\d+)?/,
      inside: {
        port: {
          pattern: /:\d+$/,
          alias: "comment",
        },
        address: {
          pattern: /[\d.]+/,
          alias: "number",
        },
      },
    },
    // Parenthetical counts like (+6)
    count: {
      pattern: /\(\+\d+\)/,
      alias: "comment",
    },
    // Percentage: 100%
    percentage: {
      pattern: /\d+%/,
      alias: "number",
    },
    // Timing: (7.8s), (0.1s)
    timing: {
      pattern: /\(\d+\.?\d*s\)/,
      alias: "comment",
    },
    // Data sizes: 13.4 KB, 180.5 KB/s
    datasize: {
      pattern: /\d+\.?\d*\s*(?:KB|MB|GB|TB)(?:\/s)?/,
      alias: "number",
    },
    // Flags: --long-flag or -s (only after whitespace)
    flag: {
      pattern: /(?<=\s)(?:--[\w-]+=?|-[a-zA-Z])\b/,
      alias: "parameter",
    },
    // Pipes and redirects
    operator: /[|><]/,
    // URLs
    url: {
      pattern: /https?:\/\/[^\s]+/,
      alias: "string",
    },
  };
}
