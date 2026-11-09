# Security policy

Report vulnerabilities privately, not in a public issue: **Security → Report a vulnerability** on this repository,
or email <nikitakarpei@mailbox.org>. Include how to reproduce it.

**In scope**: this repository: its services, libraries and distributions (the YaCy peer and the YaCy Companion app),
their builds and their container images. Flaws in the format of the signed statements go to
[Hardware-Attested Statements](https://github.com/nikitakarpei/hardware-attested-statements/security).
**Out of scope**: other YaCy peers, Android, device vendors and the services that the code relies on (such as FreeTSA
and drand), unless the code mishandles them.

We acknowledge reports within 7 days, assess them within 14, and aim to fix within 90, with the disclosure date
agreed with you. Fixes are announced in a GitHub security advisory (with a CVE where one applies); reporters are
credited unless they prefer not to be.

Only the main branch and the latest container images are supported.
