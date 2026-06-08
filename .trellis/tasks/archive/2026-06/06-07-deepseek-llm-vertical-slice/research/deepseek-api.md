# DeepSeek API Research

## Official References

* DeepSeek API quick start: https://api-docs.deepseek.com/
* DeepSeek models and pricing: https://api-docs.deepseek.com/quick_start/pricing
* DeepSeek list models API: https://api-docs.deepseek.com/api/list-models
* DeepSeek changelog: https://api-docs.deepseek.com/updates/

## Facts For First Implementation

* DeepSeek supports an API shape compatible with OpenAI and Anthropic.
* First implementation should use the OpenAI-compatible base URL:

```text
https://api.deepseek.com
```

* Current model IDs to configure:

```text
deepseek-v4-flash
deepseek-v4-pro
```

* Legacy aliases `deepseek-chat` and `deepseek-reasoner` are planned for deprecation on 2026-07-24 and should not be introduced as new project defaults.
* DeepSeek model selection must stay DB-managed through `channel_code` and `model_code`. Business usecases should not hardcode provider model IDs.

