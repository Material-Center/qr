package oicq.wlogin_sdk.request;

import java.io.Serializable;
import java.util.TreeMap;
import oicq.wlogin_sdk.sharemem.WloginSigInfo;
import oicq.wlogin_sdk.sharemem.WloginSimpleInfo;

public class WloginAllSigInfo implements Serializable {
    private static final long serialVersionUID = 1L;

    public int mainSigMap;
    public WloginSimpleInfo _useInfo;
    public TreeMap<Long, WloginSigInfo> _tk_map;
    public long _uin;
}
