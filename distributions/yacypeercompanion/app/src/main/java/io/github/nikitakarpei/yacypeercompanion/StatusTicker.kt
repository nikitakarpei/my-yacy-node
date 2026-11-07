package io.github.nikitakarpei.yacypeercompanion

import android.animation.ValueAnimator
import android.graphics.Color
import android.os.Handler
import android.os.Looper
import android.text.SpannableString
import android.text.Spanned
import android.text.style.ForegroundColorSpan
import android.widget.TextView
import io.github.nikitakarpei.yacypeercompanion.model.CompanionState

private const val TICK_MS = 500L
private const val DOT_STEPS = 3

class StatusTicker {
    private val handler = Handler(Looper.getMainLooper())
    private var state: CompanionState.Disconnected? = null
    private var line: TextView? = null
    private var step = 0

    private val tick =
        object : Runnable {
            override fun run() {
                step = (step + 1) % DOT_STEPS
                show()
                handler.postDelayed(this, TICK_MS)
            }
        }

    fun follow(state: CompanionState.Disconnected, line: TextView) {
        this.state = state
        this.line = line
        handler.removeCallbacks(tick)
        show()
        handler.postDelayed(tick, TICK_MS)
    }

    fun stop() {
        handler.removeCallbacks(tick)
    }

    private fun show() {
        val state = state ?: return
        val text = attemptTextOf(state).orEmpty()
        line?.apply {
            this.text = if (state is CompanionState.Reconnecting) withDots(text) else text
            contentDescription = text
        }
    }

    private fun withDots(text: String): CharSequence {
        val shown = if (ValueAnimator.areAnimatorsEnabled()) step + 1 else DOT_STEPS
        return SpannableString("$text...").apply {
            setSpan(ForegroundColorSpan(Color.TRANSPARENT), text.length + shown, length, Spanned.SPAN_EXCLUSIVE_EXCLUSIVE)
        }
    }
}
