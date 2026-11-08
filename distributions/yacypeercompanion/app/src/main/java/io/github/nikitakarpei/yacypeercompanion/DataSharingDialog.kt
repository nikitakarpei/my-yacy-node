package io.github.nikitakarpei.yacypeercompanion

import android.app.Activity
import android.content.Context
import android.graphics.Typeface
import android.text.SpannableStringBuilder
import android.text.Spanned
import android.text.style.BulletSpan
import android.widget.LinearLayout

private const val INTRO =
    "The developers of the app do not collect, receive or keep personal data from the app. The app sends data only to the " +
        "recipients that this text names."
private const val PEERS_INTRO =
    "The paired peer shares the index entries of each fetched page with other YaCy peers, together with this data about the device:"
private val DEVICE_DATA = listOf(
    "The Android version and the security patch dates.",
    "The operating system and its exact build.",
    "Whether the bootloader is locked.",
    "The certificates of the security hardware of the device.",
    "The app ID, the app version number and the fingerprint of the certificate that signs the app.",
    "A signing certificate that is the same on all index entries that the device signs until the app restarts.",
    "The time when the app made that certificate.",
)
private const val DEVICE_DATA_LIMITS =
    "This data does not include your name, phone number, location or the serial number of the device. The app makes a new " +
        "certificate at each start and when the certificate is about to expire."
private const val SERVICES_INTRO = "The app sends the IP address of the device to:"
private val SERVICES = listOf(
    "The websites that it fetches.",
    "The drand time service (api.drand.sh, api2.drand.sh, drand.cloudflare.com).",
    "The FreeTSA time service (freetsa.org).",
)
private const val TIME_STAMP_DATA = "FreeTSA also gets a hash of each signed page."
private const val PURPOSE =
    "The app shares this data to prove that the pages it fetches and their index entries are not modified. The app does not " +
        "use the data for advertising, analytics or profiling."
private const val CONSENT =
    "The app shares data only after you agree to it in the app. To stop sharing, unpair the device or uninstall the app."
private const val RETENTION =
    "Recipients can keep the data that the app shared without a time limit, also after you unpair the device or uninstall the app."
private const val THIRD_PARTIES =
    "The operators of YaCy peers, drand, FreeTSA and the fetched websites process the data under their own policies. App " +
        "stores can record installs and use of the app under their own policies."

fun Activity.showDataSharing(onAgree: () -> Unit) {
    showDialog(
        DATA_SHARING_TITLE,
        listOf(DialogAction("Decline"), DialogAction("Agree", onClick = onAgree)),
        content = dataSharingText(),
    )
}

private fun Context.dataSharingText(): LinearLayout = LinearLayout(this).apply {
    orientation = LinearLayout.VERTICAL
    paragraph(INTRO)
    section("Shared with Other YaCy Peers")
    paragraph(PEERS_INTRO)
    bullets(DEVICE_DATA)
    paragraph(DEVICE_DATA_LIMITS)
    section("Shared with Other Services")
    paragraph(SERVICES_INTRO)
    bullets(SERVICES)
    paragraph(TIME_STAMP_DATA)
    section("Purpose")
    paragraph(PURPOSE)
    section("Consent")
    paragraph(CONSENT)
    section("Retention")
    paragraph(RETENTION)
    section("Third Parties")
    paragraph(THIRD_PARTIES)
}

private fun LinearLayout.paragraph(text: String) {
    addView(context.bodyText(text, 14f), blockLayout(if (childCount == 0) 0 else 8))
}

private fun LinearLayout.section(title: String) {
    addView(context.heading(title, 14f).apply { setTypeface(typeface, Typeface.BOLD) }, blockLayout(16))
}

private fun LinearLayout.bullets(items: List<String>) {
    val text = SpannableStringBuilder()
    for ((index, item) in items.withIndex()) {
        val start = text.length
        text.append(item)
        if (index < items.lastIndex) text.append("\n")
        text.setSpan(
            BulletSpan(context.dp(8), context.colorOf(R.color.on_surface_variant), context.dp(2)),
            start,
            text.length,
            Spanned.SPAN_EXCLUSIVE_EXCLUSIVE,
        )
    }
    addView(context.bodyText("", 14f).apply { setText(text) }, blockLayout(4))
}

private fun LinearLayout.blockLayout(topMargin: Int) =
    LinearLayout.LayoutParams(LinearLayout.LayoutParams.MATCH_PARENT, LinearLayout.LayoutParams.WRAP_CONTENT).apply {
        this.topMargin = context.dp(topMargin)
    }
