# Web3 security: a discussion outline

> **This is a discussion outline, not a security review and not a claim of
> coverage.** F26 asked for "Web3 安全案例讨论提纲（桥攻击 / MEV / 账户抽象的防线
> 取舍）" as interview-preparation material. Each section states the attack shape,
> the defence options, and **what the trade-off costs** — because the interesting
> part of security design is rarely "add a check" and usually "choose which
> failure you can live with".
>
> Where this repository is relevant it says so explicitly, and only about
> behaviour that exists. It does not claim to defend against any of these.

Related records: [ADR-003 reorg rollback/reindex](../adr/ADR-003-reorg-rollback-reindex.md),
[ADR-005 error code taxonomy](../adr/ADR-005-error-code-taxonomy.md).

---

## 1. Bridge attacks

### Attack shape

A bridge holds value on chain A and issues a claim on chain B. The attack is
almost always at the **attestation** step, not the cryptography:

- **Forged message**: an attacker convinces the verifier that a deposit happened.
- **Replay**: a genuine message is replayed on a second chain, or twice on the
  same chain, because the message has no chain-scoped nonce.
- **Validator-set compromise**: 5-of-9 signatures are valid signatures from
  compromised keys; the maths is fine and the system is broken.
- **Upgrade abuse**: the admin path that fixes the above is itself the largest
  single point of failure.

### Defence options and their costs

| Defence | What it stops | What it costs |
| --- | --- | --- |
| Chain-scoped message IDs (include chain id + contract + nonce in the signed payload) | Cross-chain replay | A migration for existing messages; every consumer must verify the tuple, not just the hash |
| Rate limit + delay on large withdrawals | Fast draining of a compromised validator set | Legitimate large withdrawals get slow; users route around it |
| Optimistic challenge window | Forged messages, given an honest watcher | Capital lock-up for the window; the security reduces to "someone is watching" |
| Multi-sig + timelock on upgrades | Upgrade abuse | Slower incident response, which is exactly when speed matters |

### The trade-off worth arguing about

Every bridge defence trades **latency for reversibility**. A system that can
reverse a withdrawal is a system that can be pressured into reversing a valid one.
The honest position is that this is a governance question disguised as a
technical one, and the design should say which party can force a reversal.

### Where this repository is adjacent (and where it is not)

The chain-scoped identity work is the same *shape* of thinking as chain-scoped
message IDs: physical event identity includes the chain key and the block hash
precisely so the same transaction on two chains cannot collide
(`pkg/core/event_hash.go`, `pkg/plugins/database/postgres_chain_scope.go`).

**It is not a bridge defence.** This repository indexes and serves chain data; it
does not custody assets or attest cross-chain messages.

---

## 2. MEV

### Attack shape

MEV is not an exploit of a bug; it is an exploit of **ordering authority**:

- **Sandwich**: observe a pending swap, buy before it, sell after it.
- **Liquidation racing**: many bots race to be first; the winner is whoever pays
  the builder.
- **Time-bandit**: a reorg deep enough to rewrite profitable history, which
  converts consensus security into an economic question.

### Defence options and their costs

| Defence | What it changes | What it costs |
| --- | --- | --- |
| Private mempool / encrypted orderflow | Hides the trade until it is too late to sandwich | Centralises trust in whoever runs it; hurts the public mempool's transparency |
| Batch auctions | Removes intra-block ordering advantage | Latency; harder to reason about fills |
| Commit-reveal | Hides intent, then reveals | Two transactions, and the reveal is itself front-runnable if not batched |
| PBS / builder separation | Makes ordering explicit and auctioned | Introduces a builder as a trust and censorship point |

### The trade-off worth arguing about

MEV defences move the ordering decision from "whoever pays most" to "whoever the
mechanism chooses". That is not obviously better — it is a different set of
winners, and the mechanism becomes a censorship surface. The question to ask is
**who can exclude whom**.

### Where this repository is adjacent (and where it is not)

Indexers observe MEV rather than cause it: a reorg-aware indexer must not report
a sandwiched trade as if it were a settled price. The relevant machinery is
reorg handling — rollback, checkpoint rewind and derived-state invalidation
(`pkg/services/reorg/reorg_repair.go`) — and the fact that a reorged event is
auditable but excluded from default queries.

**It does not mitigate MEV** and is not in the transaction path.

---

## 3. Account abstraction

### Attack shape

AA replaces "the key signs" with "a program decides", which moves the attack
surface from the key to the program:

- **Signature-replay across accounts or chains**: a valid user operation replayed
  where it should not be.
- **Paymaster drain**: the account that sponsors gas is tricked into sponsoring
  arbitrary calls.
- **Bundler censorship / exclusion**: the bundler decides whose operation lands.
- **Validation-time state reads**: a validation function that reads mutable state
  can be manipulated between simulation and execution.

### Defence options and their costs

| Defence | What it stops | What it costs |
| --- | --- | --- |
| Nonce bound to (account, chain, validator) | Cross-account and cross-chain replay | Every validator must agree on the tuple; wallet migrations get harder |
| Single-use nonce consumed atomically | Double-spend of one operation | Requires an atomic compare-and-consume, and the store becomes a dependency |
| Paymaster allow-lists + spend caps | Drain | New integrations need approval, which is a product cost |
| Simulation + validation rules | Manipulation between sim and execution | Constrains what a validator may read, which constrains what AA can express |

### The trade-off worth arguing about

AA's promise is recoverability (social recovery, session keys, sponsored gas). Its
cost is that **"who may act for this account" becomes a program**, and programs
are upgradable. The design question is not "is the validator safe" but **who can
change the validator, and how fast**.

### Where this repository is adjacent (and where it is not)

The atomic-nonce property is the one this repository actually implements, for
SIWE rather than AA: `pkg/siwe/nonce_store.go` consumes a nonce with an atomic
compare-and-delete bound to the address, so a nonce issued to one address cannot
be consumed by another, and the acceptance is "100 concurrent consumers across two
instances, exactly one wins".

Two adjacent choices worth naming because they are the *opposite* policy to each
other, deliberately:

- the **nonce store fails closed** — with Redis down it errors rather than
  reporting success, because a false "consumed" would allow a replay
  (`TestRedisOutageFailsClosed`);
- the **query path degrades** — with the cache down it reads the source rather
  than failing, because a cache outage should cost latency, not availability
  (`pkg/services/query/degrading_reader.go`).

Choosing which of those two a given check should be is the recurring question in
all three sections above.

**It does not implement account abstraction.** There is no validator, paymaster or
bundler in this repository.

---

## 4. The common thread

Across bridges, MEV and AA the same three questions recur, and they are the ones
worth being able to answer crisply:

1. **What is the unit of identity?** (chain + contract + nonce; account + chain +
   validator; order + block) — most replay bugs are an incomplete identity.
2. **Which failure do we prefer: refusing a valid request, or accepting an
   invalid one?** Security-critical state fails closed; availability-oriented
   reads degrade.
3. **Who can change the rules, and how quickly?** Upgrades, validator sets and
   bundlers are all the same question in different clothes.
