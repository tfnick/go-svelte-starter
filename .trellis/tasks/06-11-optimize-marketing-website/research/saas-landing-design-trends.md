# SaaS Landing Design Trends Research

## Sources

* SaaSFrame, "10 SaaS Landing Page Trends for 2026": https://www.saasframe.io/blog/10-saas-landing-page-trends-for-2026-with-real-examples
* Veza Digital, "Best SaaS Landing Page Examples": https://www.vezadigital.com/post/best-saas-landing-page-examples
* Toimi, "Top 10 Best SaaS Website Designs [2026]": https://toimi.pro/blog/best-saas-website-designs/
* Saaspo bento gallery: https://saaspo.com/style/bento
* Gezar, "11 Web Design Trends in 2026": https://gezar.dk/en/blog/web-design-trends-2026
* Pravin Kumar, "Bento Grids Are Quietly Winning B2B SaaS Homepages in 2026": https://www.pravinkumar.co/blog/bento-grids-b2b-saas-homepage-design-trend-2026

## Common Patterns

Recent SaaS landing pages are moving toward:

* Product-led hero sections where the product experience is visible immediately.
* Outcome-led messaging instead of generic feature lists.
* Bento-style modules for scan-friendly value propositions.
* Stronger personality through typography, contrast, and editorial composition.
* Conversion-focused storytelling: hero, value modules, proof, pricing/CTA, objection handling, closing CTA.
* Mobile-first collapse behavior, especially for bento grids.
* Navigation and CTA systems with fewer distractions and repeated opportunities for the same primary action.

## Feasible Visual Directions

### Direction A: Product-Led Bento SaaS (Recommended)

How it works:

* Hero leads with a specific outcome and a stylized but concrete product/system preview.
* Below the hero, a bento grid maps core value propositions to B=MAP: motivation, ability, prompt.
* Pricing/product cards become the primary conversion surface.

Pros:

* Fits this starter's multiple value propositions: Go backend, Svelte app, SSR marketing, checkout, auth, events.
* Modern without needing heavy JS or animation.
* Works well with plain HTML and CSS.

Cons:

* Needs careful hierarchy so the bento grid does not become decorative noise.

### Direction B: Editorial Founder-Tool Page

How it works:

* Larger narrative sections, cream/white editorial palette, strong typography, fewer cards.
* Focuses on the story of "ship a SaaS from one binary".

Pros:

* Distinct and readable.
* Good for trust and long-form explanation.

Cons:

* May feel less immediately product-like; conversion prompts need extra care.

### Direction C: Techno-Futurist Dashboard

How it works:

* Dark, high-contrast hero with terminal/dashboard motifs, glowing product panels, dense system language.

Pros:

* Strong developer/SaaS infrastructure feel.
* Can make the embedded one-binary architecture feel powerful.

Cons:

* Easier to become generic or visually heavy.
* More likely to conflict with frontend guidance against one-note dark blue/slate or decorative effects.

## Recommendation

Use Direction A as the MVP: product-led bento SaaS with restrained color, concrete product/system panels, and conversion prompts designed through B=MAP. Borrow some editorial clarity from Direction B, but avoid making the page feel like a blog article.

The design should show the product truth early: server-rendered marketing pages, embedded Svelte app, catalog-backed pricing, checkout continuation, and production backend surfaces. This matches the trend toward real product context over abstract illustration.

## Repo Constraints

* No runtime JS is required for the marketing pages.
* CSS should remain in `marketing/assets/marketing.css`.
* Keep card radius at 8px or less.
* Avoid a one-note palette; current teal/orange/cream can be evolved with more neutral contrast and restrained accent colors.
* Use real text and product data instead of decorative screenshots that imply unavailable functionality.
