package oicq.wlogin_sdk.request;

import android.os.Parcel;
import android.os.Parcelable;
import java.nio.ByteBuffer;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.Iterator;
import java.util.Map;
import oicq.wlogin_sdk.report.c.b;
import oicq.wlogin_sdk.tools.util;

/* loaded from: classes12.dex */
public class Ticket implements Parcelable {
    public static final Parcelable.Creator<Ticket> CREATOR = new Parcelable.Creator<Ticket>() { // from class: oicq.wlogin_sdk.request.Ticket.1
        @Override // android.os.Parcelable.Creator
        public Ticket createFromParcel(Parcel parcel) {
            return new Ticket(parcel);
        }

        @Override // android.os.Parcelable.Creator
        public Ticket[] newArray(int i) {
            return new Ticket[i];
        }
    };
    private static final int EXPIRE_FIELD = 65535;
    private static final int MAX_PSKEY_SIZE = 200;
    public long _create_time;
    public long _expire_time;
    public Map<String, Long> _pskey_expire;
    public Map<String, byte[]> _pskey_map;
    public Map<String, Long> _pt4token_expire;
    public Map<String, byte[]> _pt4token_map;
    public byte[] _sig;
    public byte[] _sig_key;
    public int _type;

    public Ticket() {
        this._pskey_map = new HashMap();
        this._pskey_expire = new HashMap();
        this._pt4token_map = new HashMap();
        this._pt4token_expire = new HashMap();
    }

    public Ticket(int i, byte[] bArr, byte[] bArr2, long j, long j2) {
        this._pskey_map = new HashMap();
        this._pskey_expire = new HashMap();
        this._pt4token_map = new HashMap();
        this._pt4token_expire = new HashMap();
        this._type = i;
        this._sig = bArr == null ? new byte[0] : (byte[]) bArr.clone();
        this._sig_key = bArr2 == null ? new byte[0] : (byte[]) bArr2.clone();
        this._create_time = j;
        this._expire_time = j2;
    }

    public Ticket(int i, byte[] bArr, byte[] bArr2, long j, byte[] bArr3) {
        this._pskey_map = new HashMap();
        this._pskey_expire = new HashMap();
        this._pt4token_map = new HashMap();
        this._pt4token_expire = new HashMap();
        this._type = i;
        this._sig = bArr == null ? new byte[0] : (byte[]) bArr.clone();
        this._sig_key = bArr2 == null ? new byte[0] : (byte[]) bArr2.clone();
        this._create_time = j;
        this._expire_time = 86400 + j;
        parsePsBuf(bArr3, j, this._pskey_map, this._pskey_expire, true);
    }

    public Ticket(int i, byte[] bArr, byte[] bArr2, long j, byte[] bArr3, byte[] bArr4) {
        this._pskey_map = new HashMap();
        this._pskey_expire = new HashMap();
        this._pt4token_map = new HashMap();
        this._pt4token_expire = new HashMap();
        this._type = i;
        this._sig = bArr == null ? new byte[0] : (byte[]) bArr.clone();
        this._sig_key = bArr2 == null ? new byte[0] : (byte[]) bArr2.clone();
        this._create_time = j;
        this._expire_time = 86400 + j;
        parsePsBuf(bArr3, j, this._pskey_map, this._pskey_expire, true);
        parsePsBuf(bArr4, this._create_time, this._pt4token_map, this._pt4token_expire, false);
    }

    private Ticket(Parcel parcel) {
        this._pskey_map = new HashMap();
        this._pskey_expire = new HashMap();
        this._pt4token_map = new HashMap();
        this._pt4token_expire = new HashMap();
        readFromParcel(parcel);
    }

    private String __getPskey(String str, Map<String, byte[]> map, Map<String, Long> map2) {
        Long l;
        util.LOGI("__getPskey get domain " + str + " pskey or pt4token", "");
        if (map == null) {
            return null;
        }
        byte[] bArr = map.get(str);
        if (bArr == null) {
            util.LOGI("__getPskey get domain " + str + " pskey or pt4token null", "");
            return null;
        }
        if (map2 != null && (l = map2.get(str)) != null && l.longValue() <= u.e()) {
            util.LOGI("__getPskey delete domain " + str + " expired pskey or pt4token expire time " + l, "");
            map2.remove(str);
            map.remove(str);
            return null;
        }
        String str2 = new String(bArr);
        util.LOGI("__getPskey get domain " + str + " pskey or pt4token len " + str2.length() + " " + str2.substring(0, 5) + "***" + str2.substring(str2.length() - 5, str2.length()), "");
        return str2;
    }

    public static int calPsBufLength(Map<String, byte[]> map) {
        int i = 2;
        for (Map.Entry<String, byte[]> entry : map.entrySet()) {
            i = i + 2 + entry.getKey().length() + 2 + entry.getValue().length + 2 + 8;
        }
        return i;
    }

    private String getPskeyOrPt4tokenContent() {
        String str = "";
        for (String str2 : this._pskey_map.keySet()) {
            str = str + str2 + ":" + util.getMaskBytes(this._pskey_map.get(str2), 3, 3) + "|";
        }
        return str;
    }

    public static boolean isPskeyExpired(long j) {
        return isTicketExpired(j);
    }

    public static boolean isPskeyStorageExpired(long j) {
        long e = u.e();
        util.LOGI("isPskeyStorageExpired expireTime:" + j + "|current: " + e, "");
        return e > 86400 + j;
    }

    public static boolean isPt4TokenExpired(long j) {
        return isTicketExpired(j);
    }

    public static boolean isSkeyExpired(long j) {
        return isTicketExpired(j);
    }

    public static boolean isTicketExpired(long j) {
        long e = u.e();
        if (e > j) {
            return true;
        }
        if (j > 86400 + e) {
            util.LOGI("time for system may be  modified manually expireTime " + j + " current " + e, "");
            return true;
        }
        return false;
    }

    public static void limitMapSize(int i, Map<String, byte[]> map, Map<String, Long> map2, int i2) {
        b bVar;
        if (i > i2) {
            bVar = new b("wtlogin_alarm", "pskey_net_to_much", "");
            bVar.g.put("size", String.valueOf(i));
            util.LOGI("limitMapSize net domainCnt=" + i, "");
            map.clear();
        } else if (map.size() + i > i2) {
            bVar = new b("wtlogin_alarm", "pskey_mix_to_much", i + "," + map.size());
            bVar.g.put("size", String.valueOf(map.size() + i));
            StringBuilder sb = new StringBuilder("limitMapSize mix  domainCnt=");
            sb.append(i);
            sb.append(",localKeyMap=");
            sb.append(map.size());
            ArrayList arrayList = new ArrayList(map2.entrySet());
            Iterator it = arrayList.iterator();
            while (it.hasNext()) {
                Map.Entry entry = (Map.Entry) it.next();
                sb.append(",rm key=");
                sb.append((String) entry.getKey());
                sb.append(",expire=");
                sb.append(entry.getValue());
                map.remove(entry.getKey());
                if (map.size() <= i2 - i) {
                    break;
                }
            }
        } else {
            bVar = null;
        }
        if (bVar != null) {
            bVar.d = true;
            bVar.e = true;
        }
    }

    public static byte[] packPsBuf(Map<String, byte[]> map, long j, Map<String, Long> map2) {
        int max = Math.max(calPsBufLength(map), 4096);
        util.LOGI("packPsBuf mapSize=" + map.size() + ",bufLen=" + max, "");
        ByteBuffer allocate = ByteBuffer.allocate(max);
        allocate.putShort((short) map.size());
        for (String str : map.keySet()) {
            allocate.putShort((short) str.length());
            allocate.put(str.getBytes());
            byte[] bArr = map.get(str);
            allocate.putShort((short) bArr.length);
            allocate.put(bArr);
            allocate.putShort((short) -1);
            Long l = map2.get(str);
            allocate.putLong(l != null ? l.longValue() : 86400 + j);
        }
        allocate.flip();
        byte[] bArr2 = new byte[allocate.limit()];
        allocate.get(bArr2);
        return bArr2;
    }

    /* JADX WARN: Removed duplicated region for block: B:27:0x00b9  */
    /* JADX WARN: Removed duplicated region for block: B:30:0x00d8  */
    /* JADX WARN: Removed duplicated region for block: B:33:0x00db  */
    /* JADX WARN: Removed duplicated region for block: B:35:0x00ca  */
    /*
        Code decompiled incorrectly, please refer to instructions dump.
        To view partially-correct add '--show-bad-code' argument
    */
    public static void parsePsBuf(byte[] r22, long r23, java.util.Map<java.lang.String, byte[]> r25, java.util.Map<java.lang.String, java.lang.Long> r26, boolean r27) {
        /*
            Method dump skipped, instructions count: 280
            To view this dump add '--comments-level debug' option
        */
        throw new UnsupportedOperationException("Method not decompiled: oicq.wlogin_sdk.request.Ticket.parsePsBuf(byte[], long, java.util.Map, java.util.Map, boolean):void");
    }

    public static void parseSvrPs(byte[] bArr, long j, Map<String, byte[]> map, Map<String, Long> map2, Map<String, byte[]> map3, Map<String, Long> map4) {
        ByteBuffer byteBuffer;
        Map<String, Long> map5;
        long j2 = j;
        util.LOGI("pskeyMap " + map.size() + ", tokenMap " + map3.size() + " create time:" + j2, "");
        if (bArr == null || bArr.length <= 2) {
            return;
        }
        ByteBuffer wrap = ByteBuffer.wrap(bArr);
        short s = wrap.getShort();
        try {
            limitMapSize(s, map, map2, 200);
            limitMapSize(s, map3, map4, 200);
        } catch (Exception e) {
            util.printException(e, "");
        }
        int i = 0;
        while (i < s) {
            byte[] bArr2 = new byte[wrap.getShort()];
            wrap.get(bArr2);
            String str = new String(bArr2);
            int i2 = wrap.getShort();
            byte[] bArr3 = new byte[i2];
            wrap.get(bArr3);
            int i3 = wrap.getShort();
            byte[] bArr4 = new byte[i3];
            wrap.get(bArr4);
            short s2 = s;
            int i4 = i;
            long j3 = j2 + 86400;
            if (i2 > 0) {
                StringBuilder sb = new StringBuilder();
                sb.append("parseSvrPs add domain ");
                sb.append(str);
                byteBuffer = wrap;
                sb.append(" pskey len ");
                sb.append(i2);
                sb.append(" ");
                sb.append(j3);
                util.LOGI(sb.toString(), "");
                map.put(str, bArr3);
                map2.put(str, Long.valueOf(j3));
            } else {
                byteBuffer = wrap;
            }
            if (i3 <= 0) {
                map5 = map4;
            } else {
                String str2 = new String(bArr4);
                util.LOGI("parseSvrPs add domain " + str + " pt4token len " + i3 + " " + j3 + " " + str2.substring(0, 5) + "***" + str2.substring(str2.length() - 5), "");
                map3.put(str, bArr4);
                map5 = map4;
                map5.put(str, Long.valueOf(j3));
            }
            util.LOGI(str + " pskey:" + i2 + " pt4token " + i3 + " expire: " + j3, "");
            i = i4 + 1;
            wrap = byteBuffer;
            s = s2;
            j2 = j;
        }
    }

    @Override // android.os.Parcelable
    public int describeContents() {
        return 0;
    }

    public String getContent() {
        if (4096 == this._type) {
            return "skey:" + util.getMaskBytes(this._sig, 2, 2);
        }
        return "";
    }

    public String getPSkey(String str) {
        return __getPskey(str, this._pskey_map, this._pskey_expire);
    }

    public String getPt4Token(String str) {
        util.LOGI("getPt4Token get domain " + str + " pt4token", "");
        return __getPskey(str, this._pt4token_map, this._pt4token_expire);
    }

    public void readFromParcel(Parcel parcel) {
        this._type = parcel.readInt();
        this._sig = parcel.createByteArray();
        this._sig_key = parcel.createByteArray();
        this._create_time = parcel.readLong();
        this._expire_time = parcel.readLong();
        this._pskey_map = parcel.readHashMap(Map.class.getClassLoader());
        this._pt4token_map = parcel.readHashMap(Map.class.getClassLoader());
    }

    @Override // android.os.Parcelable
    public void writeToParcel(Parcel parcel, int i) {
        parcel.writeInt(this._type);
        parcel.writeByteArray(this._sig);
        parcel.writeByteArray(this._sig_key);
        parcel.writeLong(this._create_time);
        parcel.writeLong(this._expire_time);
        parcel.writeMap(this._pskey_map);
        parcel.writeMap(this._pt4token_map);
    }
}
