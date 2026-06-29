# api-gateway

## Public key loading

By default the gateway loads the JWT verification key from `public_key_path` in `config.yaml`.

For AWS deployment, set:

- `USE_SECRETS_MANAGER=true`
- `PUBLIC_KEY_SECRET_ID=<your-secret-name-or-arn>`

When enabled, the app loads the public key from AWS Secrets Manager instead of the local PEM file.