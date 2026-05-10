package oicq.wlogin_sdk.sharemem;

import java.io.Serializable;

public class WloginSimpleInfo implements Serializable {
    private static final long serialVersionUID = 1L;

    public byte[] _age;
    public byte[] _face;
    public byte[] _gender;
    public byte[] _img_format;
    public byte[] _img_type;
    public byte[] _img_url;
    public byte[] _nick;
    public long _uin;
    public byte[] mainDisplayName;
}
