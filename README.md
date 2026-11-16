# Vertex Relay

**Vertex Relay** is the backbone endpoint powering the full suite of Vertex services.  
It is built on the lightweight and efficient [rely](https://github.com/pippellia-btc/rely) framework.  

## Features

- **DVM Execution**: Seamlessly runs Vertex Data Vending Machines (DVMs) with low latency, ensuring a smooth experience for users.  

- **Profile Discovery**: Stores profile events like `kind:0`, `kind:3`, `kind:10000` ..., making it easy to discover and verify anyone.

- **Read-Only**: To maintain a spam-free environment, the relay does not accept direct user writes. Only our optimized [crawler](https://github.com/vertex-lab/crawler_v2) writes to the database, guaranteeing the highest-quality, most up-to-date events.

## Configuration

Configuration is read from environment variables or a `.env` file. See
[`.env.example`](.env.example) for an example.

### Relay information (NIP-11)

Operators can customize the relay information document without patching source:

| Variable | Default |
| --- | --- |
| `RELAY_NAME` | `Vertex Relay` |
| `RELAY_DESCRIPTION` | `Web of Trust Relay powered by Vertex` |
| `RELAY_ICON` | Vertex logo URL shown in `.env.example` |
| `RELAY_CONTACT` | Empty |
| `RELAY_OPERATOR_PUBKEY` | Public key derived from `RELAY_SECRET_KEY` |

`RELAY_OPERATOR_PUBKEY` accepts a hex-encoded Nostr public key (not an `npub`).
It identifies the operator in NIP-11 only. DVM responses, credit events, and
NIP-85 assertions still use `RELAY_SECRET_KEY` and its derived public key.
Clients requesting signed services must use that service identity.
Supported NIPs and software identification remain implementation-defined.

### Indexed event kinds

The relay serves events from the shared SQLite store without a metadata-kind
allowlist. Public metadata writes are rejected; ingestion is managed by
[`crawler_v2`](https://github.com/vertex-lab/crawler_v2).

In crawler v1.6.2, `FIREHOSE_KINDS` and `FETCHER_KINDS` select subscriptions and
historical fetches, but the hard-coded `pipe.ProfileKinds` also gates archival.
Expanding those environment variables alone does not archive additional kinds.
Configurable archival kinds require a change in the crawler; a relay setting
cannot enable ingestion. Existing profiles are not automatically backfilled
when the crawler's kind selection changes.

## Vertex Services

Learn more about the Vertex ecosystem and its services in our [docs](https://vertexlab.io/docs/nips/).  

## License

Licensed under MIT. For full details, see the [license](https://github.com/vertex-lab/relay/blob/master/LICENSE.md).  
