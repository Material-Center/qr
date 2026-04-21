package oicq.wlogin_sdk.report.c;

import java.util.HashMap;

/* loaded from: classes14.dex */
public class b {
    public String a;
    public String b;
    public String c;
    public String f;
    public boolean d = false;
    public boolean e = false;
    public final HashMap<String, String> g = new HashMap<>();

    public b(String str, String str2, String str3) {
        this.a = str;
        this.b = str2;
        this.c = str3;
    }

    public b a(String str) {
        this.f = str;
        return this;
    }

    public b a(String str, String str2) {
        this.g.put(str, str2);
        return this;
    }
}
