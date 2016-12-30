{{ define "username" }} 您好，{{.TargetUser.Username}}: {{end}}
{{ define "content"}} 您有一份來自用戶"{{.PkgEvent.Owner.Username}}"的新包裹"{{.PkgEvent.Pkg.Title}}"{{end}}
{{ define "remark_label"}}備註:{{end}}
{{ define "remark"}}{{.PkgEvent.Pkg.Notes}} {{end}}
{{ define "detail_label"}}詳情:{{end}}
{{ define "detail"}} 
    <p><span class="title">包裹大小:</span> {{ .PkgEvent.Pkg.HumanSize }}B</p>
    <p><span class="title">文件個數:</span> {{ .PkgEvent.Pkg.NumFiles }}</p>
    <p><span class="title">創建時間:</span> 北京時間 {{ DatetimeFormat "zh" .PkgEvent.Pkg.CreateAt }}</p>
    <p><span class="title">發送者:</span> {{ .PkgEvent.Owner.Username }} </p>
{{end}}
