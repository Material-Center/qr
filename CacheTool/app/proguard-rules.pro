# Add project specific ProGuard rules here.
# You can control the set of applied configuration files using the
# proguardFiles setting in build.gradle.
#
# For more details, see
#   http://developer.android.com/guide/developing/tools/proguard.html

# If your project uses WebView with JS, uncomment the following
# and specify the fully qualified class name to the JavaScript interface
# class:
#-keepclassmembers class fqcn.of.javascript.interface.for.webview {
#   public *;
#}

# Uncomment this to preserve the line number information for
# debugging stack traces.
#-keepattributes SourceFile,LineNumberTable

# If you keep the line number information, uncomment this to
# hide the original source file name.
#-renamesourcefileattribute SourceFile

# Gson 混淆规则
-keepattributes Signature
-keepattributes *Annotation*
-keep class sun.misc.Unsafe { *; }
-keep class com.google.gson.** { *; }
-keep class com.google.gson.stream.** { *; }

# 保护所有使用Gson序列化的类
-keep class * implements com.google.gson.TypeAdapterFactory
-keep class * implements com.google.gson.JsonSerializer
-keep class * implements com.google.gson.JsonDeserializer

# 保护TypeToken
-keep class com.google.gson.reflect.TypeToken { *; }
-keep class * extends com.google.gson.reflect.TypeToken

# 保护所有数据模型类（根据你的包名调整）
-keep class com.extracache.cachetool.model.** { *; }
-keep class com.extracache.cachetool.bean.** { *; }
-keep class com.extracache.cachetool.entity.** { *; }

# 特别保护ServerResponse和AccountRecord类
-keep class com.extracache.cachetool.model.ServerResponse { *; }
-keep class com.extracache.cachetool.model.AccountRecord { *; }
-keep class com.extracache.cachetool.model.UserDevice { *; }

# 保护网络请求相关的类
-keep class com.extracache.cachetool.service.** { *; }
-keep class com.extracache.cachetool.api.** { *; }

# 保护wlogin_sdk相关类（QQ登录SDK）
-keep class oicq.wlogin_sdk.** { *; }
-keep class oicq.wlogin_sdk.request.** { *; }
-keep class oicq.wlogin_sdk.sharemem.** { *; }
-keep class oicq.wlogin_sdk.tools.** { *; }
-keep class oicq.wlogin_sdk.report.** { *; }

# 保护oicq包下的所有类
-keep class oicq.** { *; }

# 保护OkHttp
-dontwarn okhttp3.**
-dontwarn okio.**
-keep class okhttp3.** { *; }
-keep interface okhttp3.** { *; }

# 保护MMKV
-keep class com.tencent.mmkv.** { *; }

# 保护所有使用@SerializedName注解的字段
-keepclassmembers class * {
    @com.google.gson.annotations.SerializedName <fields>;
}

# 保护所有getter和setter方法（Gson需要这些方法）
-keepclassmembers class * {
    public <methods>;
}

# 保护所有字段（防止被混淆）
-keepclassmembers class * {
    <fields>;
}