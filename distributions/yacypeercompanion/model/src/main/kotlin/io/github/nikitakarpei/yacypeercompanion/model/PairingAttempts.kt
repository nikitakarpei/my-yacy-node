package io.github.nikitakarpei.yacypeercompanion.model

sealed interface KeyDisposal {
    data object KeepKey : KeyDisposal

    data object DeleteKey : KeyDisposal

    data class UnpairAndDeleteKey(val linkText: String) : KeyDisposal
}

class PairingAttempt internal constructor(internal val linkText: String) {
    internal var givenUp = false
    internal var outcome: CompanionEvent? = null
}

class PairingAttempts {
    private var current: PairingAttempt? = null

    @Synchronized
    fun started(linkText: String): PairingAttempt = PairingAttempt(linkText).also { current = it }

    @Synchronized
    fun completed() {
        current = null
    }

    @Synchronized
    fun givenUp(): KeyDisposal? {
        val attempt = current ?: return KeyDisposal.DeleteKey
        current = null
        attempt.givenUp = true
        val outcome = attempt.outcome ?: return null
        return disposalAfter(attempt, outcome)
    }

    @Synchronized
    fun ended(attempt: PairingAttempt, outcome: CompanionEvent): KeyDisposal? {
        attempt.outcome = outcome
        if (!attempt.givenUp) return null
        return disposalAfter(attempt, outcome)
    }

    private fun disposalAfter(attempt: PairingAttempt, outcome: CompanionEvent): KeyDisposal = when {
        current != null -> KeyDisposal.KeepKey
        outcome == CompanionEvent.PairingAccepted -> KeyDisposal.UnpairAndDeleteKey(attempt.linkText)
        else -> KeyDisposal.DeleteKey
    }
}
