// Copied from attested-web-capture-android 0.2.0 (commit 75cc4a5); see its design notes there.
package io.github.nikitakarpei.httpcapture

import java.net.InetAddress
import java.net.UnknownHostException

/** IANA's special-purpose blocks that aren't "globally reachable" (checked 2026-11), plus multicast and the
 *  deprecated ::/96 and fec0::/10: https://www.iana.org/assignments/iana-ipv4-special-registry and
 *  https://www.iana.org/assignments/iana-ipv6-special-registry. ::ffff:0:0/96 needs no entry: Java turns those
 *  addresses into IPv4 ones. */
val BLOCKED = cidrs(
    "0.0.0.0/8", "10.0.0.0/8", "100.64.0.0/10", "127.0.0.0/8", "169.254.0.0/16", "172.16.0.0/12",
    "192.0.0.0/24", "192.0.2.0/24", "192.88.99.0/24", "192.168.0.0/16", "198.18.0.0/15", "198.51.100.0/24",
    "203.0.113.0/24", "224.0.0.0/4", "240.0.0.0/4",
    "::/96", "64:ff9b:1::/48", "100::/64", "100:0:0:1::/64", "2001::/23", "2001:db8::/32", "2002::/16",
    "3fff::/20", "5f00::/16", "fc00::/7", "fe80::/10", "fec0::/10", "ff00::/8",
)

/** On IPv6-only networks DNS answers with 64:ff9b::/96 addresses for IPv4-only sites; judge the IPv4 inside. */
val NAT64_PREFIX = InetAddress.getByName("64:ff9b::").address.copyOf(12)

fun isBlocked(a: InetAddress): Boolean {
    val b = a.address
    if (b.size == 16 && b.copyOf(12).contentEquals(NAT64_PREFIX)) {
        return isBlocked(InetAddress.getByAddress(b.copyOfRange(12, 16)))
    }
    return BLOCKED.matches(a)
}

fun cidrs(vararg blocks: String): List<Pair<ByteArray, Int>> =
    blocks.map { InetAddress.getByName(it.substringBefore('/')).address to it.substringAfter('/').toInt() }

fun List<Pair<ByteArray, Int>>.matches(a: InetAddress): Boolean {
    val b = a.address
    return any { (net, bits) ->
        net.size == b.size && (0 until bits).all { i ->
            val mask = 0x80 ushr (i % 8)
            (b[i / 8].toInt() and mask) == (net[i / 8].toInt() and mask)
        }
    }
}

val LOCAL_NETWORKS = cidrs(
    "127.0.0.0/8", "10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "100.64.0.0/10",
    "169.254.0.0/16", "::1/128", "fc00::/7", "fe80::/10",
)

class Refused(message: String) : UnknownHostException(message)

fun requireAllowed(host: String, addresses: List<InetAddress>) {
    for (a in addresses) if (isBlocked(a)) throw Refused("refusing non-public address ${a.hostAddress} for $host")
}
