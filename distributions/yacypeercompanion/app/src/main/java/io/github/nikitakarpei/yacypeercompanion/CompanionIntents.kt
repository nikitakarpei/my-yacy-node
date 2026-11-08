package io.github.nikitakarpei.yacypeercompanion

import android.content.Context
import android.content.Intent
import io.github.nikitakarpei.yacypeercompanion.model.CompanionEvent
import io.github.nikitakarpei.yacypeercompanion.model.PairLink

private const val ACTION_PREFIX = "io.github.nikitakarpei.yacypeercompanion."
private const val PAIR_WITH = ACTION_PREFIX + "PAIR_WITH"
private const val CANCEL_PAIRING = ACTION_PREFIX + "CANCEL_PAIRING"
private const val START = ACTION_PREFIX + "START"
private const val STOP = ACTION_PREFIX + "STOP"
private const val UNPAIR = ACTION_PREFIX + "UNPAIR"
private const val LINK_EXTRA = "link"

object CompanionIntents {
    fun intentOf(ctx: Context, event: CompanionEvent): Intent {
        val intent = Intent(ctx, CompanionHost::class.java)
        return when (event) {
            is CompanionEvent.PairWith -> intent.setAction(PAIR_WITH).putExtra(LINK_EXTRA, event.link.text)
            CompanionEvent.CancelPairing -> intent.setAction(CANCEL_PAIRING)
            CompanionEvent.StartRequested -> intent.setAction(START)
            CompanionEvent.StopRequested -> intent.setAction(STOP)
            CompanionEvent.Unpair -> intent.setAction(UNPAIR)
            else -> error("$event is not an action of the user")
        }
    }

    fun eventOf(intent: Intent): CompanionEvent? = when (intent.action) {
        PAIR_WITH -> intent.getStringExtra(LINK_EXTRA)?.let { CompanionEvent.PairWith(PairLink(it)) }
        CANCEL_PAIRING -> CompanionEvent.CancelPairing
        START -> CompanionEvent.StartRequested
        STOP -> CompanionEvent.StopRequested
        UNPAIR -> CompanionEvent.Unpair
        else -> null
    }
}

fun Context.send(event: CompanionEvent) {
    startForegroundService(CompanionIntents.intentOf(this, event))
}
