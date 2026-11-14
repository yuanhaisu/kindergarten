












<!DOCTYPE html>
<html lang="zh">

<head>

<link rel="stylesheet" type="text/css" href="../../../dist/DataTables/css/dataTables.bootstrap.min.css" />
<link rel="stylesheet" type="text/css" href="../../../dist/Buttons/css/buttons.bootstrap.min.css" />
<link rel="stylesheet" type="text/css" href="../../../dist/FixedHeader/css/fixedHeader.bootstrap.min.css" />
<link rel="stylesheet" type="text/css" href="../../../dist/Responsive/css/responsive.bootstrap.min.css" />
<link rel="stylesheet" href="../../../dist/bootstrap.css" >
<link rel="stylesheet" href="../../../dist/paging.css" >
<link rel="stylesheet" href="../../../dist/bootstrap-multiselect-0.9.13/css/bootstrap-multiselect.css" >
<link rel="stylesheet" href="../../../dist/bootstrap-datepicker-1.6.0/css/bootstrap-datepicker.min.css" >
<link rel="stylesheet" href="../../../dist/szbj.css" >
<link rel="stylesheet" href="../../../../date_new/css/data.css" />
<link rel="stylesheet" href="../../../../css/style.css" />
<link rel="stylesheet" href="../../../../js/artDialog/ui-dialog.css" />
<style>
.text-center {
    text-align: center !important;
    vertical-align: middle !important;
}

.row {
    margin-right: 15px;
    margin-left: -15px;
}
#nutritionCanvas{width:90%;height:400px;margin-left:5%;margin-right:5%}
#energyCanvas {width:100%;height:360px;}
#proteinCanvas {width:100%;height:360px;}
#fatsCanvas {width:100%;height:360px;}

@media print {
	.notPrint {
		display:none;
	}
	#nutritionCanvas{margin-left:0%}
	#analysisChart {page-break-before: always;}
	#scgnb {page-break-before: always;}
	#ssgcb {page-break-before: always;}
	#caFeFrom {page-break-before: always;}
}

@page :pseudo-class {
	size: A4 landscape;
	margin:2cm;
}

</style>
</head>

<body>
<!--计划食谱营养分析表-->
<div class="panel panel-default" style="border:none">
	<h4 id="noRecipe" class="text-center" style="display:none">该周次内没有食谱信息！
		<p class="text-center" style="height:300px"></p>
	</h4>
	<div id="hasRecipe" class="panel-body" style="display:none">
		<div id="analysisTable">
			<div><h3 class='text-center'>计划食谱营养分析表</h3></div>
			<span>&emsp;填报单位：<span id="kinderName"></span></span>
			<span class="pull-right">报表日期：<span id="reportDate"></span>&emsp;</span>
			<div>
				<table id="nutritionTable" class="table table-bordered table-hover text-center" style="width:100%">
					<thead>
						<tr><th colspan=15 class="text-center">营养摄入量</th></tr>
						<tr>
							<th></th>
							<th>热量<br>(千卡)</th>
							<th>热量<br>(千焦)</th>
							<th>蛋白质<br>(克)</th>
							<th>脂肪<br>(克)</th>
							<th>碳水化合物<br>(克)</th>
							<th>视黄醇当量<br>(微克)</th>
							<th>维生素A<br>(微克)</th>
							<th>胡萝卜素<br>(微克)</th>
							<th>维生素B1<br>(毫克)</th>
							<th>维生素B2<br>(毫克)</th>
							<th>维生素C<br>(毫克)</th>
							<th>钙<br>(毫克)</th>
							<th>锌<br>(毫克)</th>
							<th>铁<br>(毫克)</th>
						</tr>
					</thead>
					<tbody>
					</tbody>
				</table>
			</div>
			<div >
				<table id="energyTable" class="table table-bordered">
					<thead>
						<tr>
							<th rowspan=2 colspan=2 style="vertical-align: middle;">热量来源</th>
							<th colspan=2>脂肪</th>
							<th colspan=2>蛋白质</th>
							<th colspan=2>碳水化合物</th>
						</tr>
						<tr>
							<th>要求</th>
							<th>现状</th>
							<th>要求</th>
							<th>现状</th>
							<th>要求</th>
							<th>现状</th>
						</tr>
					</thead>
					<tbody>
					</tbody>
				</table>
			</div>
			<div style="float:left;width:48%">
				<table id="proteinTable" class="table table-bordered">
					<thead>
						<tr><th colspan=4 class="text-center">蛋白质来源</th></tr>
						<tr><th rowspan=2></th><th colspan=3>优质蛋白质</th></tr>
						<tr><th>要求</th><th>动物性食物</th><th>豆类</th></tr>
					</thead>
					<tbody>
					</tbody>
				</table>
			</div>
			<div style="float:right;width:48%">
				<table id="fatsTable" class="table table-bordered">
					<thead>
						<tr><th colspan=3 class="text-center">脂肪来源</th></tr>
						<tr><th></th><th>动物性</th><th>植物性</th></tr>
					</thead>
					<tbody>
					</tbody>
				</table>
			</div>
			<div class="clearfix"></div>
			<div>
				<div class="dataTable_wrapper" style="width:100%">
				<h3 class="text-center">定性分析结论</h3>
				<table id="tableAnalysis" class="table table-striped table-hover">
					<thead></thead>
					<tbody></tbody>
				</table>
				</div>
			</div>
		</div>
		<div id="analysisChart">
			<div><h3 class='text-center'>计划食谱营养分析图</h3></div>
			<div >
				<span>&emsp;填报单位：<span id="kinderName"></span></span>
				<span class="pull-right">报表日期：<span id="reportDate"></span>&emsp;</span>
			</div>
			<div id="bar">
				<div >
					<h4 class="text-center">营养摄入量</h4>
					<div id="nutritionCanvas" ></div>
				</div>
			</div>
			<div id="pie" >
				<div style="width:27%;float:left;margin-left:5%">
					<div class="panel panel-default">
						<div class="panel-heading">
							<h4 class="text-center">热量营养素来源分布（千卡）</h4>
						</div>
							<div class="panel-body text-center pie-border">
							<div id="energyCanvas"></div>
						</div>
					</div>
				</div>
				<div style="width:27%;float:left;margin-left:5%">
					<div class="panel panel-default">
						<div class="panel-heading">
							<h4 class="text-center">蛋白质来源分布（克）</h4>
						</div>
						<div class="panel-body text-center pie-border">
							<div id="proteinCanvas" ></div>
						</div>
					</div>
				</div>
				<div style="width:27%;float:right;margin-right:5%">
					<div class="panel panel-default">
						<div class="panel-heading">
							<h4 class="text-center">动物性脂肪（克）</h4>
						</div>
						<div class="panel-body text-center pie-border">
							<div id="fatsCanvas" ></div>
						</div>
					</div>
				</div>
			</div>
			<div class="clearfix"></div>
		</div>
		<div id="scgnb">
			<div><h3 class='text-center'>计划食谱三餐供能比</h3></div>
			<span>&emsp;填报单位：<span id="kinderName"></span></span>
			<span class="pull-right">报表日期：<span id="reportDate"></span>&emsp;</span>
			<div >
				<table id="scgnbWeekTable" class="table table-bordered table-hover text-center">
					<thead>
						<tr>
							<th>类别</th>
							<td>早餐</td>
							<td>早点</td>
							<td>午餐</td>
							<td>午点</td>
							<td>晚餐</td>
							<td>晚点</td>
						</tr>
					</thead>
					<tbody></tbody>
				</table>
			</div>
		</div>
		<div id="ssgcb">
			<div><h3 class='text-center'>计划食谱膳食构成表</h3></div>
			<span>&emsp;填报单位：<span id="kinderName"></span></span>
			<span class="pull-right">报表日期：<span id="reportDate"></span>&emsp;</span>
			<div >
				<table id="tableSsgcbRpt" class="table table-bordered table-hover text-center">
					<thead>
						<tr>
							<th>类别</th>
							<th>谷类</th>
							<th>薯类</th>
							<th>蔬菜</th>
							<th>水果</th>
							<th>畜禽肉类</th>
							<th>蛋类</th>
							<th>水产类</th>
							<th>大豆</th>
							<th>乳制品</th>
							<th>油脂类</th>
						</tr>
					</thead>
					<tbody></tbody>
				</table>
			</div>
		</div>
		<div id="caFeFrom">
			<div><h3 class='text-center'>计划食谱钙铁来源</h3></div>
			<span>&emsp;填报单位：<span id="kinderName"></span></span>
			<span class="pull-right">报表日期：<span id="reportDate"></span>&emsp;</span>
			<div >
				<table id="tableTglyRpt" class="table table-bordered table-hover text-center">
					<thead>
						<tr>
							<th colspan=4>钙</th>
							<th colspan=2>铁</th>
						</tr>
						<tr>
							<th>来源</th>
							<td>植物性</td>
							<td>动物性</td>
							<td>奶类及制品</td>
							<td>肉类</td>
							<td>非肉类</td>
						</tr>
					</thead>
					<tbody></tbody>
				</table>
			</div>
		</div>
	</div>
</div>














<!--[if lt IE 9]>
	<script src="/bower_components/html5shiv/dist/html5shiv.min.js?v=v1.01888"></script>
    <script src="/bower_components/respond/dest/respond.min.js?v=v1.01888"></script>
<![endif]-->

<!--<script src="/js/jquery-2.1.4.min.js?v=v1.01888"></script>-->
<!--æ´æ¢CDN-->
<script src="//apps.bdimg.com/libs/jquery/2.1.4/jquery.js"></script>

<script >
$(document).unbind('click');
</script>


<script src="/diet/dist/bootstrap.min.js?v=v1.01888"></script>


<script src="/diet/dist/bootstrap-multiselect-0.9.13/js/bootstrap-multiselect.min.js?v=v1.01888"></script>


<script src="/diet/dist/bootstrap-datepicker-1.6.0/js/bootstrap-datepicker.min.js?v=v1.01888"></script>
<script src="/diet/dist/bootstrap-datepicker-1.6.0/locales/bootstrap-datepicker.zh-CN.min.js?v=v1.01888"></script>


<script src="/diet/dist/szbj.js?v=v1.01888"></script>
<script src="/diet/dist/tipbox.js?v=v1.01888"></script>

<script type="text/javascript" src="/diet/dist/echarts-all-3.js?v=v1.01888"></script>
<script type="text/javascript" src="/diet/dist/recipe.js?v=v1.01888"></script>
<script type="text/javascript" src="/date_new/js/data.js?v=v1.01888"></script>
<script type="text/javascript" src="/js/mydatepicker/WdatePicker.js?v=v1.01888"></script>
<script type="text/javascript" src="/js/artDialog/dialog.js?v=v1.01888"></script>
<script type="text/javascript" src="/js/com/tip.js?v=v1.01888"></script>

<script type="text/javascript" src="/js/ajaxSetup.js?v=v1.01888"></script>

<script type="text/javascript" src="../../../dist/JSZip/jszip.min.js"></script>
<script type="text/javascript" src="../../../dist/DataTables/js/jquery.dataTables.min.js"></script>
<script type="text/javascript" src="../../../dist/DataTables/js/dataTables.bootstrap.min.js"></script>
<script type="text/javascript" src="../../../dist/Buttons/js/dataTables.buttons.min.js"></script>
<script type="text/javascript" src="../../../dist/Buttons/js/buttons.bootstrap.min.js"></script>
<script type="text/javascript" src="../../../dist/Buttons/js/buttons.colVis.min.js"></script>
<script type="text/javascript" src="../../../dist/Buttons/js/buttons.html5.min.js"></script>
<script type="text/javascript" src="../../../dist/Buttons/js/buttons.print.min.js"></script>
<script type="text/javascript" src="../../../dist/FixedHeader/js/dataTables.fixedHeader.min.js"></script>
<script type="text/javascript" src="../../../dist/Responsive/js/dataTables.responsive.min.js"></script>   
<script type="text/javascript">
var nutritionBarData;
var energyPieData;
var proteinPieData;
var fatsPieData;
$(document).ready(function() {
	var data = JSON.parse(sessionStorage.getItem('printData'));
	sessionStorage.removeItem('printData');
	if(data.success==true){
		if(data.body == null) {
			noRecipe.style.display = 'block';
			hasRecipe.style.display = 'none';
			return false;
		}
		noRecipe.style.display = 'none';
		hasRecipe.style.display = 'block';
		var text = formatDateRange(data.body.startDate,data.body.endDate);
		$(document).find('span[id="kinderName"]').text(data.body.orgName);
		$(document).find('span[id="reportDate"]').text(text);
		// 显示 营养摄入量
		showNutrition(data.body.nutrition);
		// 显示能量来源
		showEnergy(data.body.energy);
		// 显示 蛋白质
		showProtein(data.body.dbz);
		// 显示 脂肪来源
		showFats(data.body.fats);
		// 显示 定性分析结论
		showAnalysis(data.body.analysis);
		// 显示 三餐供能比
		showScgnbWeek(data.body.scgnb);
		// 显示 膳食构成比
		showSsgcb(data.body.ssgcb);
		// 显示 钙铁来源
		showFeAndCaFrom(data.body.tgly);
		$("#fatsTable").css("height",$("#proteinTable").outerHeight(true)-20);
		outerWidth();
		initAllchart();

		//等待页面渲染1s
		window.setTimeout(function(){
			window.print();
		},1500);
	} else {
		displayNew(data.msg);
	}
});

$(window).resize(function() {
	$("#fatsTable").css("height",$("#proteinTable").outerHeight(true)-20);
	outerWidth();
	initAllchart();
});

function outerWidth() {
	var windowWidth  = $(window).outerWidth(true);
	if(windowWidth < 1236) {
		if(windowWidth < 980) {$('body').css("font-size",'12px');}
		$('.table > thead > tr > th').css("padding",'8px 0px 8px 0px');
		$('.table > tbody > tr > th').css("padding",'8px 0px 8px 0px');
		$('.table > tfoot > tr > th').css("padding",'8px 0px 8px 0px');
		$('.table > thead > tr > td').css("padding",'8px 0px 8px 0px');
		$('.table > tbody > tr > td').css("padding",'8px 0px 8px 0px');
		$('.table > tfoot > tr > td').css("padding",'8px 0px 8px 0px');
	} else {
		if(windowWidth > 980) {$('body').css("font-size",'14px');}
		$('.table > thead > tr > th').css("padding",'8px');
		$('.table > tbody > tr > th').css("padding",'8px');
		$('.table > tfoot > tr > th').css("padding",'8px');
		$('.table > thead > tr > td').css("padding",'8px');
		$('.table > tbody > tr > td').css("padding",'8px');
		$('.table > tfoot > tr > td').css("padding",'8px');
	}
}

// 显示 营养摄入量
function showNutrition(list) {
	var trs = '';
	$.each(list,function(inde,model) {
		for(var key in model) {
			if(!isNaN(model[key])) {
				model[key] = toFixed(model[key],2);
			}
		}
		trs += '<tr><th>'+model.label+'</th><td>'+model.energy+'</td><td>'+model.heat+'</td><td>'+model.protein+'</td><td>'+
		model.fats+'</td><td>'+model.carbohydrate+'</td><td>'+model.vitaminA+'</td><td>'+model.vitaminA+'</td><td>'+
		'-'+'</td><td>'+model.vitaminB1+'</td><td>'+model.vitaminB2+'</td><td>'+model.vitaminC+'</td><td>'+
		model.ca+'</td><td>'+model.zn+'</td><td>'+model.fe+'</td></tr>';
	});
	// 显示表
	$("#nutritionTable>tbody").html(trs);
	// 显示图
	nutritionBarData = list[2];
}

//显示　能量来源
function showEnergy(list) {
	var trs = '';
	$.each(list,function(index,model) {
		for(var key in model) {
			if(!isNaN(model[key])) {
				model[key] = toFixed(model[key],2);
			}
		}
		trs += '<tr>index='+index+'<td>'+model.fatsRecommend+'</td><td>'+model.fatsIntake+'</td><td>'+model.proteinRecommend+
		'</td><td>'+model.proteinIntake+'</td><td>'+model.carbohydrateRecommend+'</td><td>'+model.carbohydrateIntake+'</td></tr>';
	});
	trs = trs.replace(/index=0/,'<th rowspan=2>摄入量</th><th>千卡</th>');
	trs = trs.replace(/index=1/,'<th>千焦</th>');
	trs = trs.replace(/index=2/,'<th colspan=2>占总热量%</th>');
	trs = trs.replace(/index=3/,'<th colspan=2>营养素比例</th>');
	$('#energyTable>tbody').html(trs);
	var model = list[0];
	energyPieData = [];
	energyPieData.push(model.fatsIntake);
	energyPieData.push(model.proteinIntake);
	energyPieData.push(model.carbohydrateIntake);
}

// 显示　蛋白质来源
function showProtein(list) {
	var trs = '';
	$.each(list,function(index,model) {
		trs += '<tr><th>'+model.label+'</th><td>'+model.require+'</td><td>'+toFixed(model.animalprotein,2)+'</td><td>'+
		toFixed(model.beanprotein,2)+'</td></tr>';
	});
	$('#proteinTable>tbody').html(trs);
	var model = list[0];
	proteinPieData = [];
	proteinPieData.push(toFixed(model.animalprotein,2));
	proteinPieData.push(toFixed(model.beanprotein,2));
	proteinPieData.push(toFixed(parseFloat(model.require)*2-parseFloat(proteinPieData[0])-parseFloat(proteinPieData[1]),2));
}

//　显示　脂肪来源
function showFats(list) {
	var trs = '';
	$.each(list,function(index,model) {
		trs += '<tr><th>'+model.label+'</th><td>'+toFixed(model.animalfats,2)+'</td><td>'+toFixed(model.vegfats,2)+'</td></tr>';
	});
	$('#fatsTable>tbody').html(trs);
	var model = list[0];
	fatsPieData = [];
	fatsPieData.push(toFixed(model.animalfats,2));
	fatsPieData.push(toFixed(model.vegfats,2));
	fatsPieData.push(toFixed(model.otherfats,2));
}

// 显示 定性分析结论
function showAnalysis(analysis) {
	$("#tableAnalysis>tbody").html('<tr><th>'+analysis+'</th></tr>')
}

// 显示 三餐供能比
function showScgnbWeek(list) {
	var trs = '';
	$.each(list,function(index,model) {
		for(var key in model) {
			if(!isNaN(model[key])) {
				model[key] = toFixed(model[key],2);
			}
		}
		trs += '<tr><th>'+model.label+'</th><td>'+model[1]+'</td><td>'+model[2]+'</td><td>'+model[4]+
		'</td><td>'+model[8]+'</td><td>'+model[16]+'</td><td>'+model[32]+'</td></tr>';
	});
	$("#scgnbWeekTable>tbody").html(trs);
}

// 显示 膳食构成表
function showSsgcb(model) {
	for(var item in model) {
		model[item] = toFixed(model[item],2);
	}
	if(true/* allHaveValue */) {
		var tr = '<tr><th>摄入量(克)</th><td>'+model['谷类']+'</td><td>'+model['薯类']+'</td><td>'+model['蔬菜']+'</td><td>'+model['水果']+'</td><td>'+model['畜禽肉类']+'</td><td>'
		+model['蛋类']+'</td><td>'+model['水产类']+'</td><td>'+model['大豆']+'</td><td>'+model['乳制品']+'</td><td>'+model['油脂类']+'</td></tr>';
		$('#tableSsgcbRpt>tbody').html(tr);
	} else {
		$('#tableSsgcbRpt>tbody').empty();
	}
}

// 显示 铁钙来源
function showFeAndCaFrom(list) {
	if(list.length > 0) {
		$.each(list,function(index,model) {
			for(var item in model) {
				if(item != 'label') {
					model[item] = toFixed(model[item],2);
				}
			}
		});
		var intakeModel = list[0];
		// 钙
		var intakeTrs = '<tr><th>'+intakeModel.label+'</th><td>'+intakeModel.vegca+'</td><td>'+
		intakeModel.animalca+'</td><td>'+intakeModel.milk+'</td>';
		// 铁
		intakeTrs += '<td class="border-left">'+intakeModel.meatfe+'</td><td>'+intakeModel.notmeatfe+'</td></tr>';
		var constituteModel = list[1];
		// 钙
		var constituteTrs = '<tr><th>'+constituteModel.label+'</th><td>'+constituteModel.vegca+'</td><td>'+
		constituteModel.animalca+'</td><td>'+constituteModel.milk+'</td>';
		// 铁
		constituteTrs += '<td class="border-left">'+constituteModel.meatfe+'</td><td>'+constituteModel.notmeatfe+'</td></tr>';
		$('#tableTglyRpt>tbody').html(intakeTrs + constituteTrs);
	} else {
		$('#tableTglyRpt>tbody').empty()
	}
}

// 加载所有图图
function initAllchart(){
	initNutritionBarData(nutritionBarData);
	initEnergyPieData(energyPieData);
	initProteinPieData(proteinPieData);
	initFatsPieData(fatsPieData);
}

// 能量来源饼图
function initEnergyPieData(list){
	var legendData = ['脂肪','蛋白质','碳水化合物']; 
	var data = [
				{value:list[0], name:'脂肪', itemStyle:{normal:{color:'#9bbb58'}}},// 绿色
                {value:list[1], name:'蛋白质', itemStyle:{normal:{color:'#c0504e'}}},// 红色
                {value:list[2], name:'碳水化合物', itemStyle:{normal:{color:'#4f81bc'}}}// 蓝色
            ];
	showPie('energy', legendData, data);
}

// 蛋白质来源饼图
function initProteinPieData(list){
	var legendData = ['动物性食物','豆类','其他'];
	var data = [
                {value:list[0], name:'动物性食物', itemStyle:{normal:{color:'#c0504e'}}},// 红色
                {value:list[1], name:'豆类', itemStyle:{normal:{color:'#9bbb58'}}},// 绿色
                {value:list[2], name:'其他', itemStyle:{normal:{color:'#4f81bc'}}}// 蓝色
            ];
	showPie('protein', legendData, data);
}

// 脂肪来源饼图
function initFatsPieData(list){
	var legendData = ['动物性','植物性','其他'];
	var data = [
                {value:list[0], name:'动物性', itemStyle:{normal:{color:'#9bbb58'}}},// 绿色
                {value:list[1], name:'植物性', itemStyle:{normal:{color:'#c0504e'}}},// 红色
                {value:list[2], name:'其他', itemStyle:{normal:{color:'#4f81bc'}}}// 红色
            ];
	showPie('fats', legendData, data);
}

// 饼图显示方法
function showPie(id, legendData, data){
    // 指定图表的配置项和数据
    var option = {
        title : { // 是否显示 标题
        	show: false
	    },
	    tooltip : { // 鼠标 指 到某个上面显示 内容 
	        trigger: 'item',
	        formatter: "{b}:{d}%({c}克)" // 格式化显示数据  {a}（系列名称），{b}（类目值），{c}（数值）
	    },
	    legend: {// 图例
	        orient: 'vertical',
	        bottom: 'bottom',
	        data: legendData
	    },
	    series : [
	        {
	            type: 'pie',
	            radius : '50%',//半径长度px
	            center: ['50%', '50%'], //中心位置[x,y]
	            data: data,
	            //roseType : 'angle',
	            itemStyle: {
	                emphasis: {
	                    shadowBlur: 10,
	                    shadowOffsetX: 0,
	                    shadowColor: 'rgba(0, 0, 0, 0.5)'
	                },
	                normal:{ 
	                    label:{ 
	                      show: true, 
	                      formatter: '{b}:\n{d}%\n({c}克)'
	                    }, 
	                    labelLine :{show:true} 
	                }
	            }
	        }
	    ]
    };
    // 使用刚指定的配置项和数据显示图表。
    window.setTimeout(function(){
    	echarts.init(document.getElementById(id+'Canvas')).setOption(option);
    }, 100);
}

// 脂肪来源饼图
function initNutritionBarData(model){
	var legendData = '';
	var data = [];
	data.push(model.energy);
	data.push(model.heat);
	data.push(model.protein);
	data.push(model.fats);
	data.push(model.carbohydrate);
	data.push(model.vitaminA);
	data.push(model.vitaminA);
	//data.push(model.carotin);
	data.push(model.vitaminB1);
	data.push(model.vitaminB2);
	data.push(model.vitaminC);
	data.push(model.ca);
	data.push(model.zn);
	data.push(model.fe);
	showBar('nutrition', legendData, data);
}

// 柱状图显示方法
function showBar(id,legendData,data) {
	var option = {
		    title: { //  显示 标题
		    	how: false
		    },
		    color: ['#5bc0de'],// 全局柱状图颜色
		    tooltip: {// 鼠标 指 到某个上面显示 内容 
		         formatter: '{a}:{c}%'// 格式化显示数据  {a}（系列名称），{b}（类目值），{c}（数值）
		    },
		    xAxis: {
		        data: ["热量(千卡)","热量(千焦)","蛋白质(克)","脂肪(克)","碳水化合物(克)"
		        ,"视黄醇当量(微克)","维生素A(微克)","维生素B1(毫克)","维生素B2(毫克)",
		        "维生素C(毫克)","钙(微克)","锌(微克)","铁(微克)"],
		        axisLabel:{interval: 0, rotate: 45}// 防止 x坐标显示丢失，但是屏幕过小会重叠
		    },
		    grid:{
		    	y2:100
		    },
		    yAxis: {
		        axisLabel : {
		                formatter: '{value} .00%'// 纵坐标格式化显示数据 
		            }
		    },
		    series: [{
		        name: '百分比',
		        type: 'bar',
		        data: data
		    }]
		};
	// 使用刚指定的配置项和数据显示图表。
    window.setTimeout(function(){
    	var chart = echarts.init(document.getElementById(id+'Canvas'));
    	chart.setOption(option);
    }, 100);
	
}
</script>
<script>
var _hmt = _hmt || [];
(function() {
  var hm = document.createElement("script");
  hm.src = "https://hm.baidu.com/hm.js?15345b55a65fc6f03c909d8dc0a401fb";
  var s = document.getElementsByTagName("script")[0]; 
  s.parentNode.insertBefore(hm, s);
})();
</script>
</body>

</html>