# Excel streaming export research

## Question

How should the backend generate Excel files for large order exports without loading all rows into memory?

## Findings

- The project currently has no Excel/XLSX dependency in `go.mod`.
- `github.com/xuri/excelize/v2` is a Go XLSX library with a documented `StreamWriter` API for writing worksheets with large amounts of data.
- Official docs state that streamed rows must be written in ascending row order and `Flush` must be called when writing is finished.
- Official docs also state that the stream writer can use temporary files when in-memory chunk data exceeds 16 MB, which matches the memory-safety requirement better than building all rows first.
- The repo uses Go 1.25.0, while current Excelize docs/package metadata require Go 1.24.0 or later, so the runtime requirement is compatible.

## Sources

- https://xuri.me/excelize/en/stream.html
- https://pkg.go.dev/github.com/xuri/excelize/v2

## Recommendation

Use `github.com/xuri/excelize/v2` with `NewStreamWriter` and row-by-row order iteration. The export worker should count rows first, reject more than 100000 rows, then stream rows to a temporary `.xlsx` file or configured object storage without constructing a full `[]Order` result.

## Implementation notes

- Add a model-level iterator or callback API for orders, such as `StreamOrders(ctx, query, batchSize, fn)`.
- Avoid offset pagination for the export body if possible; prefer stable keyset pagination using `(created_at, id)` descending to reduce skipped/duplicated rows during long exports.
- Keep the export limit check separate from the streaming loop: `CountOrders(query)` must be `<= 100000`.
- Include filters and ownership scope in the async task payload, then re-validate permissions/scope when the worker executes.
