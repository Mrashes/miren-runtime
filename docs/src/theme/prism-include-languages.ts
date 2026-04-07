import type { Prism } from "prism-react-renderer";
import siteConfig from "@generated/docusaurus.config";
import { registerMirenLanguage } from "../prism-miren";

export default function prismIncludeLanguages(PrismObject: typeof Prism) {
  const {
    themeConfig: { prism },
  } = siteConfig;
  const { additionalLanguages } = prism;

  const PrismBefore = globalThis.Prism;
  globalThis.Prism = PrismObject;

  additionalLanguages.forEach((lang: string) => {
    if (lang === "php") {
      require("prismjs/components/prism-markup-templating.js");
    }
    require(`prismjs/components/prism-${lang}`);
  });

  delete globalThis.Prism;
  if (typeof PrismBefore !== "undefined") {
    globalThis.Prism = PrismObject;
  }

  // Register our custom miren language
  registerMirenLanguage(PrismObject);
}
