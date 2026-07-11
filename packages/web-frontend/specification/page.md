# Page

A single page, `index.html`, containing a form with exactly these controls:

| Control | Element | Purpose |
| ------- | ----------------------------------------------- | ------------------------------------------------- |
| Type | `<select>` with two `<option>`s: `html`, `markdown` | The `type` to request. `html` is selected by default. |
| Spec | `<textarea>` | The natural-language specification. |
| Submit | `<button>` (label "Generate") | Submits the form. |

The `value` of each type option is the lowercase type string sent to the api
(`html` / `markdown`).

The form is labelled and laid out so each control is visibly associated with its
label. Its integration hooks are stable: the form and controls have the IDs
`form`, `type`, `spec`, and `submit` respectively, and the Type and Spec labels
use `for="type"` and `for="spec"`. The page has a clear title and heading. The
page loads `marked` from a public CDN via a `<script>` tag so that the global
`marked` is available before any generation can happen. (The CDN is reachable
only from the user's browser; the frontend's own origin serves only
`index.html`.)

## Verification

- **Required page controls are present** (`logic.test.js`) — the page has:
  - a title and a heading;
  - a form with ID `form`;
  - a type select with ID `type`;
  - selected `html` and available `markdown` options, with those lowercase
    values;
  - a Spec textarea with ID `spec`; and
  - a Generate button with ID `submit`.
