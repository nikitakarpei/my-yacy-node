package io.github.nikitakarpei.yacypeercompanion

import android.os.Handler
import android.os.Looper
import io.github.nikitakarpei.yacypeercompanion.model.CompanionState

object CompanionStore {
    private val main = Handler(Looper.getMainLooper())

    @Volatile
    var state: CompanionState = CompanionState.Stopped
        private set

    @Volatile
    var listener: ((CompanionState) -> Unit)? = null

    fun publish(state: CompanionState) {
        this.state = state
        main.post { listener?.invoke(state) }
    }
}
