# Rooch itest

To run the e2e tests, first you need to set up the devnet data:

```bash
$ make rooch-e2e-devnet
```

Then run the following command to start the e2e tests:

```bash
$ make test-e2e-rooch

# Run all tests
$ make test-e2e-rooch

# Filter specific test
$ make test-e2e-rooch-filter FILTER=TestFinalityGadget
```
