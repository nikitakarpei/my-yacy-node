import org.gradle.api.Project

const val APP_MIN_SDK = 34

fun Project.pinnedVersionOf(tool: String): String = rootProject
    .file("../../tools/tools.lock")
    .readLines()
    .map { it.trim().split(Regex("\\s+")) }
    .first { it[0] == tool }[1]
