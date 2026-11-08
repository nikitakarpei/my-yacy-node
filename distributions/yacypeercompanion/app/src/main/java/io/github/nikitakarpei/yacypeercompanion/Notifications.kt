package io.github.nikitakarpei.yacypeercompanion

import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.Context
import android.content.Intent
import android.graphics.drawable.Icon
import io.github.nikitakarpei.yacypeercompanion.model.CompanionEvent
import io.github.nikitakarpei.yacypeercompanion.model.CompanionState
import io.github.nikitakarpei.yacypeercompanion.model.DisconnectReason

const val STATUS_NOTIFICATION_ID = 1
const val FAILURE_NOTIFICATION_ID = 2
const val APP_NAME = "YaCy Companion"
private const val STATUS_CHANNEL = "status"
private const val FAILURE_CHANNEL = "failures"

fun statusNotification(ctx: Context, state: CompanionState): Notification {
    ctx
        .getSystemService(NotificationManager::class.java)
        .createNotificationChannel(NotificationChannel(STATUS_CHANNEL, "Status", NotificationManager.IMPORTANCE_LOW))
    val stop =
        PendingIntent.getService(ctx, 0, CompanionIntents.intentOf(ctx, CompanionEvent.StopRequested), PendingIntent.FLAG_IMMUTABLE)
    return Notification
        .Builder(ctx, STATUS_CHANNEL)
        .setContentTitle(APP_NAME)
        .setContentText(titleOf(state))
        .setSmallIcon(statusIconOf(state))
        .setContentIntent(homeScreenIntent(ctx))
        .addAction(Notification.Action.Builder(Icon.createWithResource(ctx, R.drawable.ic_alert), "Stop", stop).build())
        .setOngoing(true)
        .setOnlyAlertOnce(true)
        .setShowWhen(false)
        .setForegroundServiceBehavior(Notification.FOREGROUND_SERVICE_IMMEDIATE)
        .build()
}

private fun statusIconOf(state: CompanionState): Int = when (state) {
    is CompanionState.Disconnected -> R.drawable.ic_alert
    is CompanionState.Waiting -> waitingIconOf(state)
    else -> R.drawable.ic_check
}

fun waitingIconOf(state: CompanionState.Waiting): Int = when (state) {
    CompanionState.WaitingForCharger -> R.drawable.ic_plug
    CompanionState.WaitingForUnmeteredNetwork -> R.drawable.ic_wifi
}

fun postFailureNotification(ctx: Context, reason: DisconnectReason) {
    val notifications = ctx.getSystemService(NotificationManager::class.java)
    notifications.createNotificationChannel(NotificationChannel(FAILURE_CHANNEL, "Failures", NotificationManager.IMPORTANCE_DEFAULT))
    notifications.notify(
        FAILURE_NOTIFICATION_ID,
        Notification
            .Builder(ctx, FAILURE_CHANNEL)
            .setContentTitle(APP_NAME)
            .setContentText(textOf(reason))
            .setSmallIcon(R.drawable.ic_alert)
            .setContentIntent(homeScreenIntent(ctx))
            .setAutoCancel(true)
            .build(),
    )
}

fun cancelFailureNotification(ctx: Context) {
    ctx.getSystemService(NotificationManager::class.java).cancel(FAILURE_NOTIFICATION_ID)
}

private fun homeScreenIntent(ctx: Context): PendingIntent =
    PendingIntent.getActivity(ctx, 0, Intent(ctx, HomeScreen::class.java), PendingIntent.FLAG_IMMUTABLE)
