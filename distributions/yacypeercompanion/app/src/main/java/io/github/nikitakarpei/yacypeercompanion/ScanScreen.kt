package io.github.nikitakarpei.yacypeercompanion

import android.app.Activity
import android.content.Intent
import android.graphics.SurfaceTexture
import android.graphics.Typeface
import android.graphics.drawable.GradientDrawable
import android.os.Bundle
import android.view.Gravity
import android.view.TextureView
import android.view.View
import android.view.WindowInsetsController
import android.widget.FrameLayout
import android.widget.LinearLayout
import android.widget.TextView
import io.github.nikitakarpei.yacypeercompanion.model.CompanionEvent
import io.github.nikitakarpei.yacypeercompanion.model.CompanionState
import io.github.nikitakarpei.yacypeercompanion.model.PairingCode
import io.github.nikitakarpei.yacypeercompanion.model.PairingProblem
import io.github.nikitakarpei.yacypeercompanion.services.needsNewerApp
import io.github.nikitakarpei.yacypeercompanion.services.pairingCodeOf

private const val MESSAGE_MS = 3_000L
private const val RESCAN_PAUSE_MS = 1_500L
private const val SCAN_HINT_START = "The pairing code is on the "
private const val SCAN_HINT_PAGE = "Pair new device"
private const val SCAN_HINT_END = " page of the YaCy peer."

class ScanScreen : Activity() {
    private lateinit var preview: TextureView
    private lateinit var message: TextView
    private lateinit var pairingView: View
    private lateinit var scanFrame: View
    private lateinit var cameraRefusedView: View
    private var cameraAsked = false
    private var scanner: QrScanner? = null
    private var pairingCode: PairingCode? = null

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        if (!Permission.lastingGranted(this)) {
            startActivity(Intent(this, PermissionsScreen::class.java))
            finish()
            return
        }
        val screen = FrameLayout(this).apply { setBackgroundColor(0xFF16181C.toInt()) }
        preview = TextureView(this)
        screen.addView(preview, FrameLayout.LayoutParams(FrameLayout.LayoutParams.MATCH_PARENT, FrameLayout.LayoutParams.MATCH_PARENT))
        scanFrame = ScanFrame(this)
        screen.addView(scanFrame, FrameLayout.LayoutParams(dp(260), dp(260), Gravity.CENTER))
        cameraRefusedView = cameraRefusal().apply { visibility = View.GONE }
        screen.addView(cameraRefusedView, FrameLayout.LayoutParams(dp(260), dp(260), Gravity.CENTER))
        screen.addView(
            cameraOverlay(),
            FrameLayout.LayoutParams(FrameLayout.LayoutParams.MATCH_PARENT, FrameLayout.LayoutParams.MATCH_PARENT),
        )
        pairingView = pairingOverlay().apply { visibility = View.GONE }
        screen.addView(pairingView, FrameLayout.LayoutParams(FrameLayout.LayoutParams.MATCH_PARENT, FrameLayout.LayoutParams.MATCH_PARENT))
        setContentView(screen)
        window.insetsController?.setSystemBarsAppearance(0, WindowInsetsController.APPEARANCE_LIGHT_STATUS_BARS)
    }

    private fun cameraOverlay(): View = LinearLayout(this).apply {
        orientation = LinearLayout.VERTICAL
        fitsSystemWindows = true
        addView(topBar("Scan pairing code", R.color.on_camera) { finish() })
        spacer()
        addView(
            bodyText("", 16f).apply {
                text = scanHint()
                setTextColor(colorOf(R.color.on_camera_variant))
                gravity = Gravity.CENTER
                setPadding(dp(40), 0, dp(40), dp(64))
            },
        )
        message =
            bodyText("", 14f, R.color.on_surface).apply {
                visibility = View.GONE
                setPadding(dp(16), dp(14), dp(16), dp(14))
                background =
                    GradientDrawable().apply {
                        cornerRadius = dp(4).toFloat()
                        setColor(0xFFE1E2E8.toInt())
                    }
                setTextColor(0xFF191C20.toInt())
            }
        addView(
            message,
            LinearLayout.LayoutParams(LinearLayout.LayoutParams.MATCH_PARENT, LinearLayout.LayoutParams.WRAP_CONTENT).apply {
                setMargins(dp(16), 0, dp(16), dp(24))
            },
        )
    }

    private fun cameraRefusal(): View = LinearLayout(this).apply {
        orientation = LinearLayout.VERTICAL
        gravity = Gravity.CENTER
        background =
            GradientDrawable().apply {
                cornerRadius = dp(24).toFloat()
                setColor(colorOf(R.color.camera_container))
            }
        addView(icon(R.drawable.ic_camera_off, R.color.on_camera_variant, 40))
        addView(
            bodyText("No camera access", 16f, R.color.on_camera_variant),
            LinearLayout.LayoutParams(LinearLayout.LayoutParams.WRAP_CONTENT, LinearLayout.LayoutParams.WRAP_CONTENT).apply {
                setMargins(0, dp(16), 0, dp(16))
            },
        )
        addView(tonalButton("Allow camera access") { Permission.CAMERA.ask(this@ScanScreen) })
    }

    private fun scanHint(): CharSequence = android.text.SpannableStringBuilder(SCAN_HINT_START).apply {
        val start = length
        append(SCAN_HINT_PAGE)
        setSpan(android.text.style.StyleSpan(Typeface.BOLD), start, length, android.text.Spanned.SPAN_EXCLUSIVE_EXCLUSIVE)
        append(SCAN_HINT_END)
    }

    private fun pairingOverlay(): View = column().apply {
        fitsSystemWindows = true
        isClickable = true
        gravity = Gravity.CENTER_HORIZONTAL
        setPadding(dp(24), dp(24), dp(24), dp(24))
        spacer()
        addView(ProgressRing(context), LinearLayout.LayoutParams(dp(64), dp(64)))
        addView(
            heading("Pairing with the peer", 22f).apply {
                gravity = Gravity.CENTER_HORIZONTAL
                setPadding(0, dp(24), 0, dp(8))
            },
        )
        addView(bodyText("This takes a few seconds.", 14f).apply { gravity = Gravity.CENTER_HORIZONTAL })
        spacer()
        addView(textButton("Cancel") { cancelPairing() })
    }

    override fun onResume() {
        super.onResume()
        if (!::preview.isInitialized) return
        CompanionStore.listener = { state -> draw(state) }
        draw(CompanionStore.state)
    }

    override fun onPause() {
        CompanionStore.listener = null
        scanner?.stop()
        scanner = null
        super.onPause()
    }

    private fun draw(state: CompanionState) {
        when (state) {
            CompanionState.Pairing -> pairingView.visibility = View.VISIBLE
            CompanionState.Connecting, CompanionState.Connected, is CompanionState.Disconnected -> openHome()
            is CompanionState.PairingFailed -> if (pairingCode != null) showPairingFailure(state.problem) else scan()
            else -> scan()
        }
    }

    private fun scan() {
        pairingView.visibility = View.GONE
        if (!Permission.CAMERA.granted(this)) return askForCamera()
        cameraRefusedView.visibility = View.GONE
        scanFrame.visibility = View.VISIBLE
        if (scanner == null) startScanner() else scanner?.resume()
    }

    private fun askForCamera() {
        if (!cameraAsked && !Permission.CAMERA.refusedForGood(this)) {
            cameraAsked = true
            Permission.CAMERA.ask(this)
            return
        }
        scanFrame.visibility = View.GONE
        cameraRefusedView.visibility = View.VISIBLE
    }

    override fun onRequestPermissionsResult(requestCode: Int, permissions: Array<out String>, results: IntArray) {
        super.onRequestPermissionsResult(requestCode, permissions, results)
        Permission.noteAnswers(this, requestCode, results)
    }

    private fun openHome() {
        startActivity(Intent(this, HomeScreen::class.java).addFlags(Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_CLEAR_TASK))
        finish()
    }

    private fun startScanner() {
        if (preview.isAvailable) return openScanner()
        preview.surfaceTextureListener =
            object : TextureView.SurfaceTextureListener {
                override fun onSurfaceTextureAvailable(texture: SurfaceTexture, width: Int, height: Int) = openScanner()

                override fun onSurfaceTextureSizeChanged(texture: SurfaceTexture, width: Int, height: Int) = Unit

                override fun onSurfaceTextureDestroyed(texture: SurfaceTexture) = true

                override fun onSurfaceTextureUpdated(texture: SurfaceTexture) = Unit
            }
    }

    private fun openScanner() {
        if (scanner != null || isFinishing || CompanionStore.state == CompanionState.Pairing) return
        val opened = QrScanner(this, preview) { text -> runOnUiThread { pairWithPeerOf(text) } }
        scanner = opened
        runCatching { opened.start() }.onFailure { show("The camera didn’t open.") }
    }

    private fun pairWithPeerOf(text: String) {
        val code = pairingCodeOf(text)
        if (code == null) {
            show(if (needsNewerApp(text)) "This pairing code needs a newer version of the app." else "Not a pairing code of a YaCy peer.")
            preview.postDelayed({ scanner?.resume() }, RESCAN_PAUSE_MS)
            return
        }
        pair(code)
    }

    private fun pair(code: PairingCode) {
        scanner?.pause()
        pairingCode = code
        pairingView.visibility = View.VISIBLE
        send(CompanionEvent.PairWith(code))
    }

    private fun showPairingFailure(problem: PairingProblem) {
        val code = pairingCode ?: return
        pairingCode = null
        val action = if (retriesWithSameCode(problem)) DialogAction("Try again") { pair(code) } else DialogAction("OK") { cancelPairing() }
        showDialog("Couldn’t pair", listOf(action), textOf(problem), R.drawable.ic_alert) { cancelPairing() }
    }

    private fun cancelPairing() {
        pairingCode = null
        send(CompanionEvent.CancelPairing)
        scan()
    }

    private fun show(text: String) {
        message.text = text
        message.visibility = View.VISIBLE
        message.removeCallbacks(hideMessage)
        message.postDelayed(hideMessage, MESSAGE_MS)
    }

    private val hideMessage = Runnable { message.visibility = View.GONE }
}
