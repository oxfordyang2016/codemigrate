{{ define "username" }} 您好，{{.TargetUser.Username}}: {{end}}
{{ define "content"}} 您的包裹"{{.PkgEvent.Pkg.Title}}"已完成上传{{end}}
{{ define "detail_label"}}详情:{{end}}
{{ define "detail"}} 
    <p><span class="title">包裹大小:</span> {{ .PkgEvent.Pkg.HumanSize }}字节</p>
    <p><span class="title">文件个数:</span> {{ .PkgEvent.Pkg.NumFiles }}</p>
    <p><span class="title">创建时间:</span> 北京时间 {{ DatetimeFormat "zh" .PkgEvent.Pkg.CreateAt }}</p>
    <p><span class="title">完成时间:</span> 北京时间 {{ DatetimeFormat "zh" .PkgEvent.Time }}</p>
{{end}}
