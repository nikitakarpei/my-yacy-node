package io.github.nikitakarpei.servicehost

import android.Manifest
import android.annotation.SuppressLint
import android.app.Activity
import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.app.Service
import android.content.Context
import android.content.Intent
import android.content.pm.PackageManager
import android.graphics.Typeface
import android.net.Uri
import android.os.Build
import android.os.Bundle
import android.os.PowerManager
import android.provider.Settings
import android.text.InputType
import android.view.View
import android.widget.Button
import android.widget.EditText
import android.widget.LinearLayout
import android.widget.ScrollView
import android.widget.TextView
import kotlin.concurrent.thread

const val NOTIFICATION_ID = 1
const val STOP_NOTIFICATION_ID = 2
const val PREFERENCES = "service"
const val ENVIRONMENT_KEY = "environment"
const val VARIABLE_KEY_PREFIX = "variable."

interface ServiceInstance {
    fun run()

    fun stop()
}

abstract class ServiceApp(val name: String) {
    abstract val hostClass: Class<out ServiceHost>

    abstract val screenClass: Class<out ServiceScreen>

    abstract fun variables(): String

    abstract fun instanceFrom(environment: String): ServiceInstance
}

object ServiceStatus {
    @Volatile
    var running = false
        private set

    @Volatile
    var text = "Stopped"
        private set

    @Volatile
    var listener: (() -> Unit)? = null

    fun change(running: Boolean, text: String) {
        this.running = running
        this.text = text
        listener?.invoke()
    }
}

fun environmentText(ctx: Context): String =
    ctx.getSharedPreferences(PREFERENCES, Context.MODE_PRIVATE).getString(ENVIRONMENT_KEY, null).orEmpty()

fun variablesOf(template: String): List<Pair<String, String>> =
    template.lines().filter { it.isNotBlank() }.map { it.substringBefore('=') to it.substringAfter('=') }

fun saveEnvironmentText(ctx: Context, text: String) {
    ctx.getSharedPreferences(PREFERENCES, Context.MODE_PRIVATE).edit().putString(ENVIRONMENT_KEY, text).apply()
}

fun notification(ctx: Context, app: ServiceApp): Notification {
    ctx
        .getSystemService(NotificationManager::class.java)
        .createNotificationChannel(NotificationChannel("service", app.name, NotificationManager.IMPORTANCE_LOW))
    return Notification
        .Builder(ctx, "service")
        .setContentTitle(app.name)
        .setContentText("Running")
        .setSmallIcon(android.R.drawable.stat_notify_sync)
        .setContentIntent(PendingIntent.getActivity(ctx, 0, Intent(ctx, app.screenClass), PendingIntent.FLAG_IMMUTABLE))
        .setOngoing(true)
        .apply {
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.S) setForegroundServiceBehavior(Notification.FOREGROUND_SERVICE_IMMEDIATE)
        }.build()
}

fun stopNotification(ctx: Context, app: ServiceApp, reason: String): Notification {
    ctx
        .getSystemService(NotificationManager::class.java)
        .createNotificationChannel(NotificationChannel("stops", "${app.name} stops", NotificationManager.IMPORTANCE_DEFAULT))
    return Notification
        .Builder(ctx, "stops")
        .setContentTitle("${app.name} stopped")
        .setContentText(reason)
        .setStyle(Notification.BigTextStyle().bigText(reason))
        .setSmallIcon(android.R.drawable.stat_notify_error)
        .setContentIntent(PendingIntent.getActivity(ctx, 0, Intent(ctx, app.screenClass), PendingIntent.FLAG_IMMUTABLE))
        .setAutoCancel(true)
        .build()
}

abstract class ServiceHost : Service() {
    abstract val app: ServiceApp

    @Volatile
    private var instance: ServiceInstance? = null

    @Volatile
    private var destroyed = false

    private lateinit var wakeLock: PowerManager.WakeLock

    override fun onBind(intent: Intent?) = null

    @SuppressLint("WakelockTimeout")
    override fun onCreate() {
        super.onCreate()
        getSystemService(NotificationManager::class.java).cancel(STOP_NOTIFICATION_ID)
        startForeground(NOTIFICATION_ID, notification(this, app))
        wakeLock = getSystemService(PowerManager::class.java).newWakeLock(PowerManager.PARTIAL_WAKE_LOCK, "${app.name}:service")
        wakeLock.acquire()
        ServiceStatus.change(running = true, text = "Starting")
        val environment = environmentText(this)
        thread { runInstance(environment) }
    }

    private fun runInstance(environment: String) {
        val outcome =
            runCatching {
                val started = app.instanceFrom(environment)
                instance = started
                if (!destroyed) {
                    ServiceStatus.change(running = true, text = "Running")
                    started.run()
                }
            }
        ServiceStatus.change(running = false, text = outcome.exceptionOrNull()?.let { "Stopped: ${it.message}" } ?: "Stopped")
        if (!destroyed) {
            val reason = outcome.exceptionOrNull()?.message ?: "The service ended."
            getSystemService(NotificationManager::class.java).notify(STOP_NOTIFICATION_ID, stopNotification(this, app, reason))
        }
        stopSelf()
    }

    override fun onDestroy() {
        destroyed = true
        instance?.stop()
        if (wakeLock.isHeld) wakeLock.release()
        super.onDestroy()
    }
}

abstract class ServiceScreen : Activity() {
    abstract val app: ServiceApp

    private lateinit var status: TextView
    private val fields = mutableListOf<Pair<String, EditText>>()
    private lateinit var start: Button
    private lateinit var stop: Button
    private lateinit var allowNotifications: Button
    private lateinit var allowBackground: Button

    private fun notificationsMissing() = Build.VERSION.SDK_INT >= 33 &&
        checkSelfPermission(Manifest.permission.POST_NOTIFICATIONS) != PackageManager.PERMISSION_GRANTED

    private fun batteryOptimized() = !getSystemService(PowerManager::class.java).isIgnoringBatteryOptimizations(packageName)

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        val content =
            LinearLayout(this).apply {
                orientation = LinearLayout.VERTICAL
                setPadding(48, 48, 48, 48)
            }
        setContentView(
            ScrollView(this).apply {
                fitsSystemWindows = true
                addView(content)
            },
        )
        content.addView(label(app.name, 20f).apply { setTypeface(null, Typeface.BOLD) })
        status = label("", 15f).also { content.addView(it) }
        content.addView(label("Environment of the service. An empty field takes its default.", 13f))
        for ((name, default) in variablesOf(app.variables())) {
            val required = default.isEmpty()
            content.addView(
                label(if (required) "$name (required)" else name, 13f).apply {
                    setTypeface(Typeface.MONOSPACE, if (required) Typeface.BOLD else Typeface.NORMAL)
                },
            )
            fields +=
                name to
                EditText(this).apply {
                    inputType = InputType.TYPE_CLASS_TEXT or InputType.TYPE_TEXT_FLAG_NO_SUGGESTIONS
                    typeface = Typeface.MONOSPACE
                    hint = default
                    setText(rememberedValue(name))
                }.also { content.addView(it) }
        }
        start = button("Save and start") { startService() }.also { content.addView(it) }
        stop = button("Stop") { stopService(Intent(this, app.hostClass)) }.also { content.addView(it) }
        allowNotifications =
            button("Show notification") {
                startActivity(Intent(Settings.ACTION_APP_NOTIFICATION_SETTINGS).putExtra(Settings.EXTRA_APP_PACKAGE, packageName))
            }.also { content.addView(it) }
        allowBackground =
            button("Allow running in background (otherwise the service may stop while the screen is off)") {
                requestBatteryExemption()
            }.also { content.addView(it) }
        if (notificationsMissing()) {
            requestPermissions(arrayOf(Manifest.permission.POST_NOTIFICATIONS), 1)
        } else {
            askBatteryOnce()
        }
    }

    private fun rememberedValue(name: String): String =
        getSharedPreferences(PREFERENCES, Context.MODE_PRIVATE).getString(VARIABLE_KEY_PREFIX + name, null).orEmpty()

    private fun label(text: String, size: Float) = TextView(this).apply {
        this.text = text
        textSize = size
        setPadding(0, 12, 0, 12)
    }

    private fun button(text: String, onClick: () -> Unit) = Button(this).apply {
        this.text = text
        setOnClickListener { onClick() }
    }

    private fun startService() {
        val values = fields.map { (name, field) -> name to field.text.toString().trim() }
        val preferences = getSharedPreferences(PREFERENCES, Context.MODE_PRIVATE).edit()
        values.forEach { (name, value) -> preferences.putString(VARIABLE_KEY_PREFIX + name, value) }
        preferences.apply()
        saveEnvironmentText(this, values.filter { it.second.isNotEmpty() }.joinToString("") { "${it.first}=${it.second}\n" })
        ServiceStatus.change(running = true, text = "Starting")
        startForegroundService(Intent(this, app.hostClass))
    }

    override fun onRequestPermissionsResult(requestCode: Int, permissions: Array<out String>, results: IntArray) {
        super.onRequestPermissionsResult(requestCode, permissions, results)
        askBatteryOnce()
        refresh()
    }

    private fun askBatteryOnce() {
        val prefs = getSharedPreferences(PREFERENCES, Context.MODE_PRIVATE)
        if (batteryOptimized() && !prefs.getBoolean("askedBattery", false)) {
            prefs.edit().putBoolean("askedBattery", true).apply()
            requestBatteryExemption()
        }
    }

    @SuppressLint("BatteryLife")
    private fun requestBatteryExemption() {
        startActivity(Intent(Settings.ACTION_REQUEST_IGNORE_BATTERY_OPTIMIZATIONS, Uri.parse("package:$packageName")))
    }

    override fun onResume() {
        super.onResume()
        ServiceStatus.listener = { runOnUiThread { refresh() } }
        refresh()
    }

    override fun onPause() {
        ServiceStatus.listener = null
        super.onPause()
    }

    private fun refresh() {
        status.text = ServiceStatus.text
        start.isEnabled = !ServiceStatus.running
        stop.isEnabled = ServiceStatus.running
        allowNotifications.visibility = if (notificationsMissing()) View.VISIBLE else View.GONE
        allowBackground.visibility = if (batteryOptimized()) View.VISIBLE else View.GONE
    }
}
