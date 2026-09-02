# localhostrunagent configuration

The agent opens one localhost.run tunnel, forwards the tunnel traffic to the
node, and leases the public hostname to the node process. It runs as a sidecar
of one node.

## Environment variables

| Variable | Default | Meaning |
| --- | --- | --- |
| `LOCALHOSTRUN_AGENT_ADDRESS` | none, required | The address of this container on the node network. |
| `YACY_NODE_ORIGIN` | `http://yacy-rwi-node:8090` | The HTTP origin of the node. |
| `PROCESS_ENVIRONMENT_LEASE_SOCKET` | `/run/yacy/process-environment-lease.sock` | The Unix socket that carries the lease. |
| `LOCALHOST_RUN_HOST` | `localhost.run` | The tunnel provider host. |
| `LOCALHOST_RUN_IDENTITY_FILE` | none | The SSH private key file. |
| `LOCALHOST_RUN_KNOWN_HOSTS` | `/state/known_hosts` | The persistent host key file. |

Give the container a static address on the node network and set
`LOCALHOSTRUN_AGENT_ADDRESS` to that address. The agent sends every request to
the node from this address. Set `YACY_TRUSTED_PROXIES` of the node to the same
address with a `/32` or `/128` prefix. The node then accepts a forwarded client
address from the agent only.

## Hostname stability

Without an identity file, the agent connects as `nokey` and localhost.run does
not promise the same hostname for a new tunnel. Register an SSH public key at
`https://admin.localhost.run/` and set `LOCALHOST_RUN_IDENTITY_FILE` to the
private key file to keep one hostname.

## Volumes

Mount a persistent volume at `/state`. It holds the host key that the agent
accepts on the first connection. A later host key change stops the agent.

Mount the directory of the lease socket in the node container at the same path.
Both containers must have access to this directory.

## Lease

After the tunnel opens, the agent grants `YACY_ADVERTISE_HOST` and
`YACY_ADVERTISE_PORT` to the node process. The node entrypoint waits for this
grant before it starts the node.

The agent stops when the tunnel, the ingress, or the lease consumer stops. Use a
restart policy on both containers to get a new tunnel.

## Restart pace

An agent that stops in its first 10 seconds holds its exit until the 10 seconds
have passed. The provider refuses a tunnel that comes too soon after the last
one, so a container that restarts many times each second makes its own failure
worse. An agent that ran longer stops at once. A signal also stops it at once.
