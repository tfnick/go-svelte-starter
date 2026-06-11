# Fogg Behavior Model Research

## Sources

* Fogg Behavior Model official site: https://www.behaviormodel.org/
* Prompts in the Fogg Behavior Model: https://www.behaviormodel.org/prompts
* Ability in the Fogg Behavior Model: https://www.behaviormodel.org/ability
* Motivation in the Fogg Behavior Model: https://www.behaviormodel.org/motivation

## Key Takeaways

The official model states that behavior happens when Motivation, Ability, and a Prompt come together at the same time. If the target behavior does not happen, at least one of the three elements is missing.

The official prompts guidance also maps "Prompt" to familiar product terms such as cue, trigger, call to action, and request. It recommends matching prompt type to the user's motivation/ability context and warns against asking for a behavior that is too complicated too early.

For marketing website optimization, the target behavior should be concrete. In this project, likely target behaviors are:

* Primary: visitor clicks a plan CTA and enters `/app/checkout?product_id=...`.
* Secondary: visitor opens `/app` to explore or sign in.
* Tertiary: visitor understands the product enough to continue to `/features` or `/pricing`.

## Mapping to Marketing Pages

### Motivation

Increase motivation by making the value emotionally and practically clear:

* State the concrete pain: shipping a SaaS requires marketing, auth, checkout, admin, events, and deployment glue.
* State the outcome: one Go binary with server-rendered marketing pages and a Svelte app.
* Add credible proof cues: production-oriented features, SEO endpoints, embedded deployment, product-catalog-driven pricing.
* Use outcome-led headings rather than generic feature names.
* Use Fogg's motivation axes lightly: pain/pleasure for current shipping pain, hope/fear for faster launch vs brittle glue code, belonging/rejection for building like production SaaS teams.

### Ability

Increase ability by making the next action easier:

* Keep the primary CTA visible and specific.
* Explain the checkout path in 2-3 simple steps.
* Reuse product catalog cards so users do not need to decode plans manually.
* Keep page sections scan-friendly with short labels, visible hierarchy, and clear mobile collapse.
* Avoid requiring app login before the user understands why to proceed.
* Favor tools/resources and smaller next steps over "educating" users. For example, show the launch path and plan cards instead of asking users to infer architecture from long copy.

### Prompt

Prompts are the page triggers that ask the visitor to act now:

* Hero CTA: pick a plan / view pricing.
* Pricing card CTA: start checkout.
* Section-level CTA after explaining value.
* Final CTA after objection handling.

Prompt type should match user state:

* High motivation + low ability: facilitator prompt, e.g. "See the 3-step launch path".
* Low motivation + high ability: spark prompt, e.g. stronger outcome or risk-reduction copy.
* High motivation + high ability: signal prompt, e.g. direct "Start checkout".

## Recommended Page Strategy

Use the home page as a behavior chain:

1. Hook motivation with a specific outcome.
2. Raise ability with a simple "how it works" path.
3. Support motivation with proof and feature/outcome modules.
4. Trigger action with pricing/product CTA.

The pricing page should be lower-friction and action-heavy. The features page should support evaluation, not become a disconnected feature catalog.

## Repo Constraints

* The current implementation uses server-rendered Go templates, so all behavior design must be expressed in static HTML structure plus server data.
* `html/template` should remain because product names, descriptions, URLs, and JSON-LD are dynamic and need safe escaping.
* Product CTAs already flow to `/app/checkout?product_id=...`; this is the strongest existing conversion prompt.
