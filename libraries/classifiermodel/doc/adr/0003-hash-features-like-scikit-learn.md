# 3. Hash features like scikit-learn

Date: 2026-09-29

## Status

Accepted

## Context

The trainer fits the spam model with scikit-learn. Spamproxy scores pages
in Go. Each feature must go to the same index in both.

## Decision

- `featurehashing` hashes each feature with MurmurHash3 like the
  `HashingVectorizer` of scikit-learn.
- The trainer computes feature rows with this library through its
  `spamfeatures` program, and fits them with scikit-learn.

## Consequences

Spamproxy needs no Python. Golden tests keep the hashes equal to the
hashes of scikit-learn.
