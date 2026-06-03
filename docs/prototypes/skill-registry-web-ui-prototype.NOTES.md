# Skill Registry Web UI Prototype Notes

Question:

> What should the first Web UI for the current `Registry Core MVP` look like?

Scope:

- three variants of the same throwaway web console
- switchable via `?variant=control|studio|gallery`
- screens switchable via `?screen=overview|catalog|import|detail|audit`

Run:

```bash
python3 -m http.server 4173 -d docs/prototypes
```

Then open:

- <http://localhost:4173/skill-registry-web-ui-prototype.html>

Variant intent:

- `control`
  - dense operations console for internal platform teams
- `studio`
  - workflow-first packaging/import studio for publishers
- `gallery`
  - calmer marketplace-style catalog for broad readers

Prototype constraints:

- throwaway code
- static data only
- no real API calls
- built to answer layout / hierarchy / workflow questions, not production readiness
