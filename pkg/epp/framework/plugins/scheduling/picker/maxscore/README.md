# Max Score Picker

**Type:** `max-score-picker`

Selects the endpoint(s) with the highest score calculated during the scoring phase.

> [!NOTE]
> This plugin is enabled by default if no other picker is specified. You do not need to explicitly declare it in your configuration.

## What it does

1.  Receives a list of `ScoredEndpoint` candidates.
2.  Shuffles the list in-place to ensure random tie-breaking when multiple endpoints share the same maximum score.
3.  Sorts the candidates by score in descending order.
4.  Forms a top tier from candidates whose score meets `topScoreRatio * maximum score`.
5.  Randomizes candidates within that tier.
6.  Returns up to `maxNumOfEndpoints` candidates. If the tier is smaller than that limit, the picker fills the remaining positions in descending score order to preserve multi-endpoint behavior.

## Behavioral Intent

This picker maximizes the adherence to scoring objectives (e.g., cache affinity, lowest load). However, it is susceptible to **hot-spotting** if many concurrent requests produce identical scores for the same endpoint (e.g., identical prompts targeting a specific cache hit).

## Inputs consumed

- Consumes the list of `ScoredEndpoint` results from the scoring phase.

## Configuration

The plugin config supports:

- `maxNumOfEndpoints` (default 1)
  - The maximum number of endpoints to pick and return. Must be > 0. If more candidates are available than this limit, the picker returns a random subset of the top tier.
- `topScoreRatio` (default 1)
  - Randomly selects from endpoints whose score is at least this fraction of the maximum score. The value must be greater than 0 and at most 1. A value of 1 restricts selection to endpoints tied for the maximum score.

> [!TIP]
> In most production scenarios, `maxNumOfEndpoints` is left at its default value of `1` to select a single target endpoint for the request.

To reduce hot-spotting while retaining score locality, select one endpoint randomly from the top score tier:

```yaml
- type: max-score-picker
  name: top-tier-picker
  parameters:
    maxNumOfEndpoints: 1
    topScoreRatio: 0.95
```
