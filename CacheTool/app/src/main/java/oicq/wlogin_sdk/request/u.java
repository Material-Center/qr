package oicq.wlogin_sdk.request;

import android.content.Context;
import java.net.Socket;
import java.security.SecureRandom;

/* loaded from: classes12.dex */
public class u {
    public static int A;
    public static int B;
    public static int C;
    public static byte[] D;
    public static byte[] E;
    public static int F;
    public static byte[] G;
    public static byte[] H;
    public static byte[] I;
    public static byte[] J;
    public static byte[] K;
    public static byte[] L;
    public static byte[] M;
    public static byte[] N;
    public static byte[] O;
    public static byte[] P;
    public static byte[] Q;
    public static byte[] R;
    public static int S;
    public static int T;
    public static int U;
    public static int V;
    public static int W;
    public static int X;
    public static int Y;
    public static boolean Z;
    public static byte[] a0;
    public static long b0;
    public static long c0;
    public static byte[] d0;
    public static boolean e0;
    public static int f0;
    public static byte[] g0;
    public static byte[] h0;
    public static byte[] i0;
    public static byte[] j0;
    public static String l0;
    public static long n0;
    public static Object p0;
    public static boolean q0;
    public static String r0;
    public static int s0;
    public static SecureRandom t;
    public static String t0;
    public static Boolean u;
    public static String u0;
    public static boolean v;
    public static int v0;
    public static Context w;
    public static String w0;
    public static int x;
    public static String y;
    public static int z;
    public byte[] a = null;
    public byte[] b = new byte[16];
    public long d = 0;
    public String e = "";
    public long f = 0;
    public int g = 0;
    public int i = 0;
    public int j = 5000;
    public int k = 0;
    public byte[] l = new byte[16];
    public byte[] m = new byte[16];
    public int n = 1;
    public byte[] o = new byte[0];
    public int p = 0;
    public Socket q = null;
    public Socket r = null;
    public int s;

    static {
        SecureRandom secureRandom;
        try {
            secureRandom = new SecureRandom();
        } catch (Throwable th) {
            secureRandom = null;
        }
        t = secureRandom;
        u = Boolean.FALSE;
        v = true;
        w = null;
        x = 2052;
        y = "";
        z = 0;
        A = 1;
        B = 0;
        C = 0;
        D = new byte[0];
        E = new byte[0];
        F = 0;
        G = new byte[0];
        H = new byte[0];
        I = new byte[0];
        J = new byte[0];
        K = new byte[0];
        L = "android".getBytes();
        M = new byte[0];
        N = new byte[0];
        O = new byte[0];
        P = new byte[0];
        Q = new byte[0];
        R = new byte[0];
        S = 0;
        T = 0;
        U = 0;
        V = 0;
        W = 0;
        X = 0;
        Y = 0;
        Z = false;
        a0 = new byte[0];
        b0 = 0L;
        c0 = 0L;
        d0 = new byte[4];
        e0 = false;
        f0 = 1;
        h0 = new byte[0];
        i0 = new byte[0];
        j0 = new byte[0];
        l0 = "";
        n0 = 0L;
        p0 = new Object();
        q0 = false;
        s0 = 0;
    }

    public static long e() {
        return System.currentTimeMillis() / 1000;
    }
}
