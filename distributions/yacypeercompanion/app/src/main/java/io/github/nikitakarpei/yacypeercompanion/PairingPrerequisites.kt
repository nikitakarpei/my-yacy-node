package io.github.nikitakarpei.yacypeercompanion

import android.content.Context

object PairingPrerequisites {
    fun met(ctx: Context): Boolean = Permission.lastingGranted(ctx) && DataSharing.agreed(ctx)
}
