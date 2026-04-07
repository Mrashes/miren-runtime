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
    hint: {
      pattern: /^\(.*\)$/m,
      alias: "comment",
    },
    "status-success": {
      pattern: /✓.*/,
      alias: "inserted",
    },
    "status-progress": {
      pattern: /[⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏].*/,
      alias: "comment",
    },
    "progress-bar": {
      pattern: /[░▒▓█]{2,}/,
      alias: "builtin",
    },
    "deploy-url-label": {
      pattern: /Your app is available at:/,
      alias: "builtin",
    },
    "version-id": {
      pattern: /\b\w+-v[A-Za-z0-9]{10,}\b/,
      alias: "constant",
    },
    arrow: {
      pattern: /→/,
      alias: "operator",
    },
    // Matched before miren-full so ALL_CAPS words in table output
    // don't get consumed as miren subcommands
    "table-header": {
      pattern: /\b[A-Z]{2,}\b/,
      alias: "tag",
    },
    selector: {
      pattern: /▸/,
      alias: "keyword",
    },
    // Captures [sudo] miren <command> [subcommand] as a single match,
    // then the `inside` grammar splits it into distinct token types
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
    sudo: {
      pattern: /\bsudo\b/,
      alias: "keyword",
    },
    // Split IP:port so the port can be styled muted like the real CLI
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
    count: {
      pattern: /\(\+\d+\)/,
      alias: "comment",
    },
    percentage: {
      pattern: /\d+%/,
      alias: "number",
    },
    timing: {
      pattern: /\(\d+\.?\d*s\)/,
      alias: "comment",
    },
    datasize: {
      pattern: /\d+\.?\d*\s*(?:KB|MB|GB|TB)(?:\/s)?/,
      alias: "number",
    },
    // Lookbehind ensures we don't match hyphens inside words like "sample-apps"
    flag: {
      pattern: /(?<=\s)(?:--[\w-]+=?|-[a-zA-Z])\b/,
      alias: "parameter",
    },
    operator: /[|><]/,
    url: {
      pattern: /https?:\/\/[^\s]+/,
      alias: "string",
    },
  };
}
