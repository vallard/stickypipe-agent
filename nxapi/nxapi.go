package nxapi

/* These structures show the type of response we get back from
The NXAPI.  This could be quite big.  The Output is the same up until the
body.  That is where the outputs differ depending on which command is given.
*/
type NXAPI_Response struct {
	Ins_api Ins_API
}

type Ins_API struct {
	Type    string
	Version string
	Sid     string
	Outputs map[string]Output
}

type Output struct {
	Input string
	Msg   string
	Code  string
	Body  map[string]interface{}
}

type Version struct {
	Header_Str        string
	Bios_Ver_Str      string
	Kickstart_Ver_Str string
	Bios_Cmpl_Time    string
	Kick_File_Name    string
	Kick_Cmpl_Time    string
	Kick_Tmstmp       string
	Chassis_Id        string
	Cpu_Name          string
	Memory            int
	Mem_Type          string
	Proc_Board_Id     string
	Host_Name         string
	Bootflash_Size    int
	Kern_Uptm_Days    int
	Kern_Uptm_Hrs     int
	Kern_Uptm_Min     int
	Kern_Uptm_Secs    int
	Rr_Reason         string
	Rr_Sys_Ver        string
	Rr_Service        string
	Manufacturer      string
}

type InterfaceCounters struct {
}
