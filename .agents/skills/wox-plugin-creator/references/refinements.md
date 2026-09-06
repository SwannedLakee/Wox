# Query Refinements

Return query-scoped chips such as type filters or sort modes on `QueryResponse.Refinements`. Wox renders the controls and sends the selected values back on the next query as `Query.Refinements` / `query.refinements`. The plugin owns the filtering or sorting semantics.

Requires `MinWoxVersion` >= `2.0.4` (or `2.4.2` for single-file SDK plugins). Do not invent command syntax such as `type:file` when a refinement can express the same control.

## Hotkey format (required)

Refinement hotkeys use the platform primary modifier:

- macOS: `cmd+<key>` (for example `cmd+t`)
- Windows and Linux: `ctrl+<key>` (for example `ctrl+t`)

Detect the OS at runtime and emit one of those real hotkey strings. Do not write a literal `ctrl/cmd+t` token. Wox does not parse that form.

```typescript
const primaryHotkey = (key: string) =>
  (process.platform === "darwin" ? "cmd+" : "ctrl+") + key;

{ Hotkey: primaryHotkey("t") }
```

```python
import sys

def primary_hotkey(key: str) -> str:
    modifier = "cmd" if sys.platform == "darwin" else "ctrl"
    return f"{modifier}+{key}"

QueryRefinement(..., hotkey=primary_hotkey("t"))
```

Pick a mnemonic letter (`t` for type, `s` for sort or status). Leave `Hotkey` empty only when the control should have no shortcut.

### Reserved shortcuts

Do not reuse Wox launcher chords:

- `cmd+j` / `ctrl+j` — action panel
- `cmd+f` / `ctrl+f` — open the refinements/filters panel
- `enter` and `cmd+enter` / `ctrl+enter` — default and alternate result actions

## Read selected values

Selected values are opaque strings keyed by refinement `Id`. Multi-select values are comma-separated.

- Node.js: `query.Refinements?.["item_type"] ?? "all"`
- Python: `query.refinements.get("item_type", "all")`

## Node.js example

```typescript
const primaryHotkey = (key: string) =>
  (process.platform === "darwin" ? "cmd+" : "ctrl+") + key;

async query(ctx, query): Promise<QueryResponse> {
  const selectedType = query.Refinements?.["item_type"] ?? "all";
  return {
    Results: this.filterResults(selectedType),
    Refinements: [
      {
        Id: "item_type",
        Title: "Type",
        Type: "singleSelect",
        Hotkey: primaryHotkey("t"),
        DefaultValue: ["all"],
        Options: [
          { Value: "all", Title: "All" },
          { Value: "file", Title: "Files" },
          { Value: "folder", Title: "Folders" },
        ],
      },
    ],
  };
}
```

## Python example

```python
import sys
from wox_plugin import QueryRefinement, QueryRefinementOption, QueryRefinementType, QueryResponse

def primary_hotkey(key: str) -> str:
    modifier = "cmd" if sys.platform == "darwin" else "ctrl"
    return f"{modifier}+{key}"

def build_type_refinement() -> QueryRefinement:
    return QueryRefinement(
        id="item_type",
        title="Type",
        type=QueryRefinementType.SINGLE_SELECT,
        hotkey=primary_hotkey("t"),
        default_value=["all"],
        options=[
            QueryRefinementOption(value="all", title="All"),
            QueryRefinementOption(value="file", title="Files"),
            QueryRefinementOption(value="folder", title="Folders"),
        ],
    )

async def query(self, ctx, query) -> QueryResponse:
    selected_type = query.refinements.get("item_type", "all")
    return QueryResponse(
        results=self.filter_results(selected_type),
        refinements=[build_type_refinement()],
    )
```
