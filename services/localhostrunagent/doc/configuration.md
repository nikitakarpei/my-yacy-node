# localhostrunagent configuration

The agent opens one localhost.run tunnel, forwards the tunnel traffic to the
node, and leases the public hostname to the node process. It runs as a sidecar
of one node. A supervisor must restart both processes.

## Environment variables

| Variable | Default | Meaning |
| --- | --- | --- |
| `LOCALHOSTRUN_AGENT_ADDRESS` | none, required | Static address the agent binds. It sends every request to the node from this address. |
| `YACY_NODE_ORIGIN` | `http://yacy-rwi-node:8090` | The HTTP origin of the node. |
| `PROCESS_ENVIRONMENT_LEASE_SOCKET` | `/run/yacy/process-environment-lease.sock` | The Unix socket that carries the lease. |
| `LOCALHOST_RUN_HOST` | `localhost.run` | The tunnel provider host. |
| `LOCALHOST_RUN_IDENTITY_FILE` | none | The SSH private key file. |
| `LOCALHOST_RUN_KNOWN_HOSTS` | `/state/known_hosts` | The persistent host key file. |

Set `YACY_TRUSTED_PROXIES` of the node to `LOCALHOSTRUN_AGENT_ADDRESS` with a
`/32` or `/128` prefix.

## Hostname stability

Without an identity file, localhost.run replaces the hostname every 20 minutes.
Register an SSH public key at `https://admin.localhost.run/` and set
`LOCALHOST_RUN_IDENTITY_FILE` for a hostname that lasts longer. A stable
hostname needs a Custom Domain plan.

## Storage

Give the agent a persistent `/state` directory for the host key it accepts on
the first connection.

The node reads the lease socket, so give the node the same path.
