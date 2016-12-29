{{ define "username" }} 您好，{{.TargetUser.Username}}: {{end}}
{{ define "content"}} 您有一份来自用户"{{.PkgEvent.Owner.Username}}"的新包裹"{{.PkgEvent.Pkg.Title}}"{{end}}
{{ define "remark_label"}}备注:{{end}}
{{ define "remark"}}{{.PkgEvent.Pkg.Notes}} {{end}}
{{ define "detail_label"}}详情:{{end}}
{{ define "detail"}} 
    <p><span class="title">包裹大小:</span> {{ .PkgEvent.Pkg.HumanSize }}B</p>
    <p><span class="title">文件个数:</span> {{ .PkgEvent.Pkg.NumFiles }}</p>
    <p><span class="title">创建时间:</span> 北京时间 {{ DatetimeFormat "zh" .PkgEvent.Pkg.CreateAt }}</p>
    <p><span class="title">发送者:</span> {{ .PkgEvent.Owner.Username }} </p>
{{end}}
