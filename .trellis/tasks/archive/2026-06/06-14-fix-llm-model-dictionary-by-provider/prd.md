# Fix LLM Model Dictionary By Provider

## Background

Parameter -> LLM currently uses a single `llm_model` dictionary for all LLM providers. DeepSeek and SiliconFlow support different model sets, so a shared model dropdown can show invalid options for the selected provider.

## Goal

Make the model dropdown in Parameter -> LLM resolve options from the selected adapter/provider schema, with provider-specific dictionary types maintained in the backend dictionary system.

## Scope

- Add provider-specific LLM model dictionary types for DeepSeek and SiliconFlow.
- Expose the model dictionary type from each LLM adapter schema.
- Update `Parameters.svelte` to load and render model options using the current schema's model dictionary type.
- Clear incompatible selected model values when the provider/adapter changes.
- Add regression coverage for the schema contract.

## Data Flow

```text
Svelte Parameters form
  -> listParameterIntegrationSchemas('llm')
  -> schema.model_dictionary_type
  -> getDictionaries([model_dictionary_type, ...schema field dictionaries])
  -> model dropdown options
  -> create/update parameter integration channel payload
```

## Acceptance Criteria

- DeepSeek LLM schema returns a DeepSeek-specific model dictionary type.
- SiliconFlow LLM schema returns a SiliconFlow-specific model dictionary type.
- The LLM model dropdown changes when the selected provider/adapter changes.
- The frontend does not hardcode provider-specific model options.
- Existing embedding/provider configuration behavior remains compatible.

## Verification

- `go test ./api/usecase`
- `go test ./api/routes`
- `cd frontend && npm test`
- `cd frontend && npm run build`
