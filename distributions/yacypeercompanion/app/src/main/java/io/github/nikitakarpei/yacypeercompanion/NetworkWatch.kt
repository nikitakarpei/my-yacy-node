package io.github.nikitakarpei.yacypeercompanion

import android.content.Context
import android.content.SharedPreferences
import android.net.ConnectivityManager
import android.net.Network
import android.net.NetworkCapabilities
import android.net.NetworkRequest
import android.util.Log
import io.github.nikitakarpei.yacypeercompanion.model.NetworkAccess
import java.util.concurrent.ConcurrentHashMap

private const val LOG_TAG = "NetworkWatch"

class NetworkWatch(private val ctx: Context, private val onChange: (NetworkAccess) -> Unit) : AutoCloseable {
    private data class NetworkTraits(val validated: Boolean, val unmetered: Boolean)

    private val connectivity = ctx.getSystemService(ConnectivityManager::class.java)
    private val networks = ConcurrentHashMap<Network, NetworkTraits>()

    @Volatile
    private var reportedAccess: NetworkAccess? = null

    val access: NetworkAccess
        get() {
            val traits = networks.values.ifEmpty { listOfNotNull(activeNetworkTraits()) }
            return when {
                traits.isEmpty() -> NetworkAccess.NONE
                meteredNetworksAllowed.isOn(ctx) || traits.any { it.unmetered } -> NetworkAccess.USABLE
                else -> NetworkAccess.METERED_ONLY
            }
        }

    private val callback =
        object : ConnectivityManager.NetworkCallback() {
            override fun onAvailable(network: Network) {
                if (networks.putIfAbsent(network, NetworkTraits(validated = false, unmetered = false)) ==
                    null
                ) {
                    report("available: $network")
                }
            }

            override fun onCapabilitiesChanged(network: Network, capabilities: NetworkCapabilities) {
                val traits = traitsOf(capabilities)
                if (networks.put(network, traits) != traits) report("changed: $network, $traits")
            }

            override fun onLost(network: Network) {
                networks.remove(network)
                report("lost: $network")
            }
        }

    private val meteredNetworksSetting =
        SharedPreferences.OnSharedPreferenceChangeListener { _, _ -> if (access != reportedAccess) report("metered networks setting") }

    init {
        val everyNetwork =
            NetworkRequest
                .Builder()
                .removeCapability(NetworkCapabilities.NET_CAPABILITY_INTERNET)
                .removeCapability(NetworkCapabilities.NET_CAPABILITY_NOT_VPN)
                .build()
        connectivity.registerNetworkCallback(everyNetwork, callback)
        meteredNetworksAllowed.follow(ctx, meteredNetworksSetting)
    }

    private fun report(change: String) {
        val access = access
        reportedAccess = access
        Log.d(LOG_TAG, "network $change, access: $access")
        onChange(access)
    }

    private fun activeNetworkTraits(): NetworkTraits? = connectivity.getNetworkCapabilities(connectivity.activeNetwork)?.let {
        traitsOf(it)
    }

    private fun traitsOf(capabilities: NetworkCapabilities) = NetworkTraits(
        validated = capabilities.hasCapability(NetworkCapabilities.NET_CAPABILITY_VALIDATED),
        unmetered = capabilities.hasCapability(NetworkCapabilities.NET_CAPABILITY_NOT_METERED),
    )

    override fun close() {
        meteredNetworksAllowed.unfollow(ctx, meteredNetworksSetting)
        connectivity.unregisterNetworkCallback(callback)
    }
}
