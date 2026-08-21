# 0006. Treat weak authentication as first-class, policy-gated Nuclei work

- Status: accepted
- Date: 2026-08-20

## Context

The important internet exposure triad for Anchor is high-risk service/port exposure, high-risk vulnerability evidence, and weak authentication. Nuclei tags already provide broad protocol coverage, while RBKD maintains custom templates used by the worker image. Applying a generic default-login tag to services such as SSH can lock accounts or create uncontrolled authentication traffic. Anchor must expose the coverage without owning or silently expanding the template-maintenance scope.

## Decision

Weak-authentication work is routed through Nuclei tags and approved custom sources. Standard campaigns may run anonymous, no-auth, empty-credential, and explicitly safe checks. High lockout-risk services, including SSH, exclude the official `default-login` tag and use the approved RBKD custom-template policy instead. Account-locking checks require explicit campaign authorization, protocol-specific attempt and target budgets, lockout/429/423 detection, cancellation, and redacted evidence. Successful credentials are never persisted as plaintext or reused for follow-on work.

RBKD owns template content and maintenance. Anchor owns source and revision/digest recording, activation state, routing policy, task provenance, scope/budget enforcement, and result normalization. Template bundles must be promoted from a traceable revision or digest; an unpinned branch snapshot is not a release input.

## Consequences

Two-high-one-weak coverage becomes visible and auditable without adding a separate password engine to Anchor. Template updates remain an external coordination point and require compatibility fixtures, source/license review, and a worker smoke test. A campaign that does not authorize lockout-risk checks must report the resulting coverage gap rather than imply that weak authentication was assessed.
