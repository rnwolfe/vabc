// @ts-check
import { defineConfig } from "astro/config";
import starlight from "@astrojs/starlight";
import starlightLlmsTxt from "starlight-llms-txt";

// Canonical site URL. Currently the live Vercel alias; cut over to https://vabc.sh
// once the domain is bought (update here + redeploy + re-alias + regenerate OG cards).
const SITE = process.env.SITE_URL || "https://vabc-cli.vercel.app";

export default defineConfig({
  site: SITE,
  trailingSlash: "ignore",
  integrations: [
    starlight({
      title: "vabc",
      description:
        "Virginia ABC product search and store inventory from your terminal — agent-friendly, read-only.",
      tagline: "Find the bottle. Skip the website.",
      plugins: [starlightLlmsTxt()],
      logo: { src: "./src/assets/mark.svg", replacesTitle: false },
      customCss: ["./src/styles/tokens.css", "./src/styles/docs.css"],
      social: { github: "https://github.com/rnwolfe/vabc" },
      editLink: { baseUrl: "https://github.com/rnwolfe/vabc/edit/main/site/" },
      head: [
        {
          tag: "meta",
          attrs: { property: "og:image", content: SITE + "/og/default.png" },
        },
        {
          tag: "meta",
          attrs: { name: "twitter:image", content: SITE + "/og/default.png" },
        },
        { tag: "meta", attrs: { name: "twitter:card", content: "summary_large_image" } },
      ],
      sidebar: [
        { label: "Start here", autogenerate: { directory: "docs/start" } },
        { label: "Guides", autogenerate: { directory: "docs/guides" } },
        { label: "Reference", autogenerate: { directory: "docs/reference" } },
      ],
    }),
  ],
});
