package io.github.nikitakarpei.yacypeercompanion

import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.content.IntentFilter
import android.content.SharedPreferences
import android.os.BatteryManager
import android.util.Log
import io.github.nikitakarpei.yacypeercompanion.model.PowerAccess

private const val LOG_TAG = "PowerWatch"

class PowerWatch(private val ctx: Context, private val onChange: (PowerAccess) -> Unit) : AutoCloseable {
    @Volatile
    private var pluggedIn = pluggedInNow()

    @Volatile
    private var reportedAccess: PowerAccess? = null

    val access: PowerAccess
        get() = if (!ChargingOnly.required(ctx) || pluggedIn) PowerAccess.USABLE else PowerAccess.CHARGER_NEEDED

    private val charger =
        object : BroadcastReceiver() {
            override fun onReceive(context: Context, intent: Intent) {
                pluggedIn = intent.action == Intent.ACTION_POWER_CONNECTED
                report(intent.action.orEmpty())
            }
        }

    private val chargingOnlySetting = SharedPreferences.OnSharedPreferenceChangeListener { _, _ -> report("charging only setting") }

    init {
        val chargerChanges =
            IntentFilter().apply {
                addAction(Intent.ACTION_POWER_CONNECTED)
                addAction(Intent.ACTION_POWER_DISCONNECTED)
            }
        ctx.registerReceiver(charger, chargerChanges, Context.RECEIVER_NOT_EXPORTED)
        ChargingOnly.follow(ctx, chargingOnlySetting)
        reportedAccess = access
    }

    private fun pluggedInNow(): Boolean {
        val battery = ctx.registerReceiver(null, IntentFilter(Intent.ACTION_BATTERY_CHANGED))
        return (battery?.getIntExtra(BatteryManager.EXTRA_PLUGGED, 0) ?: 0) != 0
    }

    private fun report(change: String) {
        val access = access
        if (access == reportedAccess) return
        reportedAccess = access
        Log.d(LOG_TAG, "power $change, access: $access")
        onChange(access)
    }

    override fun close() {
        ChargingOnly.unfollow(ctx, chargingOnlySetting)
        ctx.unregisterReceiver(charger)
    }
}
