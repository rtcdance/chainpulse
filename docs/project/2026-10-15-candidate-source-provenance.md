# Candidate source provenance — 2026-10-15

This is an implementation record for the Approved T00–T18 delivery, not a completion report. The candidate starts at `30c45b05485b4ecfb47ae5accd3b723c09fad4e1`. The original working directory and all four historical stashes remain intact. The managed candidate receives a hash-checked snapshot; generated executables (including the root `loadtest` binary) are excluded.

Existing dirty changes are inherited inputs, not claimed as newly authored work. They provide configuration, finality strategy, generation/outbox, test isolation and acceptance tooling used by this delivery. Their inclusion follows the approved full-plan scope; unrelated stash cleanup and local tooling are not applied. Baseline equality describes this capture only, not future edits.

The current implementation still needs production finality observations, canonical PostgreSQL hot-cache integration, native subscription reorg propagation, the specified load/soak measurements, immutable images and actual rollback, remote same-candidate CI/CD, and final evidence collection. `Approved` must not become `Implemented` on the basis of this snapshot.

## Inherited dirty inputs

| Path | Compared with initial task baseline | Candidate SHA-256 |
| --- | --- | --- |
| `.github/workflows/ci-integration.yml` | subsequently updated | `f7a0083b698beca93c962f15809357a7dd7a6f6c6d08ab16ffd2924475ae6623` |
| `.github/workflows/ci.yml` | subsequently updated | `2b6792e916fa018a3aa04f41fa8430aa69b7bc0bab917f30b84b5e869a7789d2` |
| `.github/workflows/coverage.yml` | subsequently updated | `d50c7c46bd74e42c1d57b5cd4da52bf47fd74f48ba423e4a112dbf670c0073ec` |
| `.github/workflows/frontend.yml` | subsequently updated | `51598124941d2f962adad86c8a7c61da98dbcc29b847e6ff198ffe8afa3760b1` |
| `.github/workflows/metrics.yml` | subsequently updated | `376845ddae74af654068a916cdff51192e5b891646ad50a0d09d27e623efc5ff` |
| `.github/workflows/test.yml` | subsequently updated | `af288260c3b4c4f83f37463213a42f6d67709363ba61221d34815bdb5d87ac43` |
| `.gitignore` | byte-identical | `da2aa797b523cb7139cba13364b7247edc8aa159b84bce2c5f4462e02ba1cef8` |
| `README.md` | subsequently updated | `3cee9b5f424fd894634484aa6187173618922b8afcc7fb8ce27a69378852e0fa` |
| `cmd/microservices/puller/main.go` | subsequently updated | `4519ac58698ae9914ad5e73b6e0ad24e2be50eb4c6dba418d4039417046ef6fb` |
| `cmd/microservices/puller/puller_execution.go` | subsequently updated | `e74182db2120041776e5339394b2e5f6051fc286e28e44973e7b18400f54b2c4` |
| `cmd/monolithic/chainpulse/m1a_runtime_wiring.go` | subsequently updated | `e5d463314bf5e2145e67dd9bc3e113415605193c20c1591edd519301ad763b68` |
| `cmd/monolithic/chainpulse/main.go` | subsequently updated | `63f3389c5726805b41d5587ec156c55dbbeaf30c02cc3fd99204d4ac42f0b29f` |
| `pkg/application/bootstrap/core_config.go` | byte-identical | `2f3669df4ba80652954d8d51c251d630731e04ad9ea3e043fcb53882b44e20b7` |
| `pkg/application/bootstrap/indexing_storage.go` | subsequently updated | `feb9f218fadc3865acf0d89e7346d378b22ab1cfc232b5b791d990d70224d5ff` |
| `pkg/application/bootstrap/monolithic_starter.go` | subsequently updated | `1dd82c0a18cb423931213d33d2c4e224858aae78b7d52e19fd59cb32792e96df` |
| `pkg/configmodel/config.go` | byte-identical | `6ba129c870e35fcc3f10e65138313c707ac603a14e67a1cbc2ee8cdf7644eb77` |
| `pkg/core/config.go` | byte-identical | `8e2b2cda5b1ed35ce401fe8e19edd742af104f63042b7d796263020bb3d00304` |
| `pkg/infrastructure/deployment/deployment_config_property_test.go` | byte-identical | `4e578a4cc8406467a8c2f9c9a5eee0015bc5125aea8a8cd3102db413ddc9e60e` |
| `pkg/plugins/api/admin_api_key_handler.go` | subsequently updated | `9ffa8da062201122ee50baf8f7c209fa45d1dd3be196ec2f6979dd964f1c49ec` |
| `pkg/plugins/api/cors_middleware.go` | subsequently updated | `f7eebdb3ffd81acd0ca5f6510e8af6ec075599bb9201e643f2e66da79c01b6ef` |
| `pkg/plugins/api/cors_middleware_test.go` | byte-identical | `94b6a964eb9fc9d80eadcc89df565fde3563a178a805d291605d867094e2c269` |
| `pkg/plugins/api/rate_limiter.go` | subsequently updated | `3767a4d6939f15507bbec898b159ef30c70201c707ce48217ef44472cada6024` |
| `pkg/plugins/api/rate_limiter_test.go` | subsequently updated | `49b1c3a4600f01a7496cb61d7d4baacb5334600992d607810c6a915927859b3b` |
| `pkg/plugins/api/response.go` | subsequently updated | `f0dd38446972e8f9da37eabd9c96ee714f7a77ca264b63ff7d466bcd35e5e65d` |
| `pkg/plugins/api/webhook_store.go` | subsequently updated | `772f8c8864c11e0673f9d769ae38a886682e90026e445eb35bd10c4e1a7b60c0` |
| `pkg/plugins/database/postgres_reorg_outbox.go` | byte-identical | `d213e6dba4a21b2ea030e46d99062c47a66104dfca8fe76d1579c13bc9e33fb9` |
| `pkg/plugins/pullers/data_puller.go` | subsequently updated | `3a25d42f8cbc922bbe3919cd1632cec5def60d2460a6691f2dfed523745ae16e` |
| `pkg/plugins/pullers/data_puller_base_test.go` | byte-identical | `e485e18ba032232332eae23275a76b597be6062e2fbf50548545a4f5c4a20ca6` |
| `pkg/plugins/pullers/https_jsonrpc_puller.go` | subsequently updated | `492e92b1e3fcb46ca86dd4215a49be85de0d93af48baaced3fbc9bdda0822f9a` |
| `pkg/plugins/pullers/multi_chain_puller.go` | subsequently updated | `ed6a53229fa3145b503443d42d4f057d9ac650a50de7fced2934b843e6482536` |
| `pkg/plugins/pullers/multi_chain_puller_test.go` | byte-identical | `fb9c205125b959cd521edc11571874d39aa6a3778d02ac51e5855bd317a3db3a` |
| `pkg/plugins/pullers/puller_helpers_test.go` | byte-identical | `2ec3fc1a7003356def3dd5651fa9e3d1cc194c83aee52c8e70970a6502e25285` |
| `pkg/services/indexing/multi_chain_indexer.go` | subsequently updated | `57c84334ac054cd646e0e27ed89af718f3dbde70a24ef766f2e0ab7261317e25` |
| `pkg/services/processor/batch_committer_test.go` | subsequently updated | `32819d9f01ee4aa0d727ed7d525be0ad203ad1ce977819c06dc97b0e9759ecdf` |
| `pkg/services/reorg/reorg_handler.go` | subsequently updated | `b7b6ff30172bf54df8babe34211c4c43fb827465ff87b4549df7a81c4b208acd` |
| `test/acceptance/kafka_redelivery_test.go` | subsequently updated | `02a1e4957b6c4fa7aa685fc18f29fcb3e07988aece12c2fcd210553f15b58199` |
| `test/acceptance/skip-allowlist.json` | subsequently updated | `411112c954ea83b89ac6b1284ea85c3e6b4f1b2fb78af2dfab698ef6e3ba3479` |
| `test/e2e/data_puller_test.go` | byte-identical | `c7871447846a27eeb658c5de43f54ded8452bdbc42dbc0aa67fe9f314eb24001` |
| `test/e2e/database_manager_property_test.go` | byte-identical | `2c2fafb0b0cea2b0b252881e4b9f8a5055bec6acf86219c165b809eb539eba58` |
| `test/e2e/database_manager_test.go` | byte-identical | `f9f2261970cfb5fd028ad0d46a006ec5b84f4d5dc2546ea8f66abc4de1634e3b` |
| `test/e2e/error_scenarios_test.go` | byte-identical | `0bc1f21dc2a3c49adc58867a4ad2a3208378eca448146d4cd5c2742f63003bfb` |
| `test/e2e/happy_path_scenarios_test.go` | byte-identical | `cdf59fccf8d3d72c54887596af629f1022d03d350b59bfc66bd695987dd0d286` |
| `test/e2e/indexer_manager.go` | byte-identical | `bb0d5e7b127994712b5e2ae4062f7fe57cd8ac1aec9eaef05c02897bfb218e82` |
| `test/e2e/multi_chain_scenarios_test.go` | byte-identical | `75e453aa96054acb07ca257667f465a94f9bb757db0cc0fe4f2ef44ea3c3dda1` |
| `test/e2e/orchestrator.go` | byte-identical | `0708f9b4202e98057db0aa1db2b53dc877ba417c0e3ee2b170604e9f95685440` |
| `test/e2e/orchestrator_test.go` | byte-identical | `8d54ee503005139884fe006abeb5edf6a4dfecccf6f5a1d47d6391d118958f46` |
| `test/e2e/performance_manager.go` | byte-identical | `5117ab933a4329a78ca0844ee5ceace8fc01363d167174b1c8be092ec56a7fdb` |
| `test/e2e/performance_manager_property_test.go` | byte-identical | `b861c67388780809da5a0b6cdf30f39a8afbb309b09e951c908f040ae1f6e22c` |
| `test/e2e/performance_scenarios_test.go` | byte-identical | `49e02b7976f6965f8f3b12f16a0168b321314b8ff50aec4c45583db3612b5b04` |
| `migrations/000014_reorg_generations.down.sql` | byte-identical | `96a0f3d60d8dff347ed1b2cadde71367bfd4d6d7ba0bae1ab30751e4f69d7bbe` |
| `migrations/000014_reorg_generations.up.sql` | byte-identical | `4bf37a89ded45c16a6f46c91cf30cde77fd3f3168e2f60e7a90cffbde2c1f5d5` |
| `pkg/application/bootstrap/alerts.go` | subsequently updated | `573f6c58366b3396dedbe4c746de37e3b79f1ba9cf1aa7218c495456cac6b9f7` |
| `pkg/application/bootstrap/alerts_test.go` | byte-identical | `aa2945fa1ab66fab3c42e9535430a58a7e2e50dc62979e6684808d608389c019` |
| `pkg/application/bootstrap/indexing_committer.go` | subsequently updated | `d6e9a72a3e5d4a339fc518fb99a0a3577e86f1ee3ac92ae83b036f80b5994bf2` |
| `pkg/application/bootstrap/indexing_committer_test.go` | subsequently updated | `5351012ac0d4ae5f5072cf6154cc30dc7ff36dc01752e3dbd6c1cf9309ede01d` |
| `pkg/plugins/api/error_redaction_test.go` | subsequently updated | `74b89d0f9eee6fc32ccde77b02395c7b333c57608321e39daf1e08f1677b12dc` |
| `pkg/plugins/api/operator_rate_limit_test.go` | subsequently updated | `35a138d9d92781e6e6cb712cf7596bcb9e5f89554a0260163e2a9c384133a2ca` |
| `pkg/plugins/pullers/https_jsonrpc_puller_finality_test.go` | byte-identical | `f68a69e81dc1df72f1022e275dd760a7ed2f65f0591e2b2faeb1fff61ab3715c` |
| `pkg/services/finality/strategy.go` | byte-identical | `5cc9ea87844226b62f7f0d6ed2e4a8e9ddce960c70a585ccb274f7902e849e24` |
| `pkg/services/finality/strategy_test.go` | byte-identical | `52241e97a90e97f2ebc12c7b1f4c45932c03af0408ad6e5592c31fedd7079cbe` |
| `pkg/services/processor/batch_committer_drain_gate_test.go` | subsequently updated | `a797b5e4699e6615c88201013e61da63ddf83ee9a3d8aed530ff8009056ad5ca` |
| `pkg/services/reorg/generation.go` | byte-identical | `8941459f3f0b3a0b865605ad5608bd80bd1d04075b1b66a2c90a3f0ed852a949` |
| `pkg/services/reorg/generation_test.go` | byte-identical | `df0f6dc89be97979aa42a7c1feae4e2236f4b92696065cc014e97769ee1d331d` |
| `pkg/services/reorg/reorg_digest_test.go` | subsequently updated | `d6d2ea9b015d1ac0b5679ec9e1aa5dbb90011beef801492407434a3a055e3c49` |
| `pkg/services/reorg/service.go` | subsequently updated | `601e468207caa61bde818c0ea845915c2d2878eeccd5effae61cb8817e7464b8` |
| `pkg/services/reorg/service_test.go` | subsequently updated | `1ba680614d36a5468353e9288360bbe298ab1fe036b0e04f217aa2fd28e09c9a` |
| `pkg/siwe/verify_store_negative_test.go` | byte-identical | `f83ac172ab9f150fc1ddbf9f3b14d46a9e174055fc5c1ed1164676e2c9ce79a8` |
| `scripts/verify-production-readiness.sh` | byte-identical | `b1dd54bf7b11f89623cf59a84948f22439daad6db59e92a12798b1db2aaf74d6` |
| `test/benchmarks/cache_baseline.json` | byte-identical | `00b43b32176bd6f9041679ac387958a2a380e7e29f61e2657968d99fe3864581` |
| `test/benchmarks/core_baseline.json` | byte-identical | `b3440e3686d8b72f3054b393bece5e1c5d83eb0e2f653c557c28e656e03cb896` |
| `test/benchmarks/evm_baseline.json` | byte-identical | `bf59574c83cd95302a9cbaaf4a7e59d49a8e6b375aa3a67eedb62a4a996efe7e` |
| `test/benchmarks/pullers_baseline.json` | byte-identical | `f66c66586c05d96e7a23b7ceff1dc3293e7e0593b7d1d1d34ee25ca29748e8ee` |
| `test/benchmarks/reorg_baseline.json` | byte-identical | `7153cc5ee8e0af01a2fa77c3b283581d6777a0040be91c74a49d30c6d429d055` |
| `test/e2e/database_env_test.go` | byte-identical | `68cbf6eba39dc66afa31bcd7830323c6dca636e61ff09fed9805c2abcd35add8` |

The detailed machine inventory and raw execution logs are retained in the task evidence directory; these dirty local runs are not final clean-SHA release proof. CI check display names are now sourced from `test/acceptance/required-checks.json`. The server must confirm the corresponding active ruleset before T01 can be verified.
