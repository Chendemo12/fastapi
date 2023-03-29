package fastapi

import (
	"context"
	"fmt"
	"github.com/Chendemo12/fastapi/logger"
	"time"
)

type CronJob interface {
	// String 可选的任务文字描述
	String() string
	// Interval 任务执行间隔, 调度协程会在此时间间隔内循环触发 Do 任务, 任务的触发间隔不考虑任务的执行时间
	Interval() time.Duration
	// Do 定时任务, 每 Interval 个时间间隔都将触发此任务,即便上一个任务可能因超时未执行完毕.
	// 其中 Do 的执行耗时应 <= Interval 本身, 否则将视为超时, 超时将触发 WhenTimeout
	Do(ctx context.Context) error
	// WhenError 当 Do 执行失败时触发的回调, 若 Do 执行失败且超时, 则 WhenError 和 WhenTimeout
	// 将同时被执行
	WhenError(errs ...error)
	// WhenTimeout 当定时任务未在规定时间内执行完毕时触发的回调, 当上一次 Do 执行超时时, 此 WhenTimeout 将和
	// Do 同时执行, 即 Do 和 WhenError 在同一个由 WhenTimeout 创建的子协程内。
	WhenTimeout()
}

type Job struct{}

// String 任务文字描述
func (c *Job) String() string { return "Job" }

// Interval 执行调度间隔
func (c *Job) Interval() time.Duration { return 5 * time.Second }

// Do 定时任务
func (c *Job) Do() func(ctx context.Context) error {
	fmt.Printf("%s Run at %s\n", c.String(), time.Now().String())
	return nil
}

// WhenTimeout 当任务超时时执行的回调
func (c *Job) WhenTimeout() {
	fmt.Printf("%s Timeout at %s\n", c.String(), time.Now().String())
}

// WhenError 当 Do 执行失败时触发的回调
func (c *Job) WhenError(errs ...error) {
	return
}

type Schedule struct {
	job         CronJob
	pctx        context.Context
	ctx         context.Context
	ticker      *time.Ticker
	cancel      context.CancelFunc
	logger      logger.Iface
	msgDisabled bool
}

func (s *Schedule) output(p func(args ...any), args ...any) {
	if s.msgDisabled {
		return
	}
	p(args...)
}

// String 任务描述
func (s *Schedule) String() string { return s.job.String() }

// AtTime 到达任务的执行时间
func (s *Schedule) AtTime() <-chan time.Time { return s.ticker.C }

// Do 执行任务
func (s *Schedule) Do() {
	done := make(chan struct{}, 1)
	ctx, cancel := s.ctx, s.cancel // 当下一次调度时，此被修改
	defer cancel()                 // 执行完毕/超时，关闭子协程
	defer close(done)

	go func() {
		err := s.job.Do(ctx)
		done <- struct{}{} // 任务执行完毕

		if err != nil { // 此次任务执行发生错误
			s.output(s.logger.Warn, fmt.Sprintf("'%s' error occur: %s", s.job.String(), err.Error()))
			s.job.WhenError(err)
		}
	}()

	select {
	case <-done:
		return
	case <-time.After(s.job.Interval()):
		s.output(s.logger.Warn, fmt.Sprintf("'%s' do timeout", s.job.String()))
		// 单步任务执行时间超过了任务循环间隔,认为超时
		s.job.WhenTimeout()
		return
	}
}

// Cancel 取消此定时任务
func (s *Schedule) Cancel() {
	s.cancel()
	s.ticker.Stop()
}

// Scheduler 当时间到达时就启动一个任务协程
func (s *Schedule) Scheduler() {
	for {
		// 每次循环都将创建一个新的 context.Context 避免超时情况下互相影响
		s.ctx, s.cancel = context.WithTimeout(s.pctx, s.job.Interval())
		select {
		case <-s.pctx.Done(): // 父节点被关闭,终止任务
			return
		case <-s.AtTime(): // 到达任务的执行时间, 创建一个新的事件任务
			go s.Do()
		}
	}
}

type Scheduler struct {
	ctx         context.Context
	logger      logger.Iface
	schedules   []*Schedule
	msgDisabled bool
}

func (s *Scheduler) SetLogger(logger logger.Iface) *Scheduler {
	s.logger = logger
	return s
}

// DisableMsg 禁用内部日志输出
func (s *Scheduler) DisableMsg() *Scheduler {
	s.msgDisabled = true
	return s
}

// QuerySchedule 依据任务描述查找任务
func (s *Scheduler) QuerySchedule(title string) *Schedule {
	for i := 0; i < len(s.schedules); i++ {
		if s.schedules[i].String() == title {
			return s.schedules[i]
		}
	}

	return nil
}

// AddCronjob 添加任务
func (s *Scheduler) AddCronjob(jobs ...CronJob) *Scheduler {
	for _, job := range jobs {
		j := &Schedule{
			job:         job,
			pctx:        s.ctx, // 绑定父节点Context
			ticker:      time.NewTicker(job.Interval()),
			logger:      s.logger,
			msgDisabled: s.msgDisabled,
		}
		s.schedules = append(s.schedules, j)
	}

	return s
}

func (s *Scheduler) Add(jobs ...CronJob) *Scheduler { return s.AddCronjob(jobs...) }

// Run 启动任务调度器（非阻塞）
func (s *Scheduler) Run() {
	for i := 0; i < len(s.schedules); i++ {
		s.schedules[i].msgDisabled = s.msgDisabled // 使之后的修改生效
		s.schedules[i].logger = s.logger

		if !s.msgDisabled {
			s.logger.Debug(fmt.Sprintf("Cronjob: '%s' started.", s.schedules[i].String()))
		}
		go s.schedules[i].Scheduler()
	}
}

func (s *Scheduler) Done() <-chan struct{} { return s.ctx.Done() }

// NewScheduler 创建一个任务调度器
//
//	@param	ctx	context.Context	根Context
//	@param	lg	logger.Iface	可选的日志句柄
//	@return	scheduler 任务调度器
//
//	# Usage
//	// 首先创建一个根 Context
//	pCtx, _ := context.WithTimeout(context.Background(), 50*time.Second)
//	scheduler := NewScheduler(pCtx, logger)
//
//	// 定义任务, 需实现 CronJob 接口
//	type Clock struct {
//		Job	// 默认 Job 未实现 Do 方法
//		lastTime time.Time
//	}
//
//	func (c *Clock) Interval() time.Duration { return 1 * time.Second }
//	func (c *Clock) String() string          { return "报时器" }
//
//	func (c *Clock) Do(ctx context.Context) error {
//		diff := time.Now().Sub(c.lastTime)
//		c.lastTime = time.Now()
//		fmt.Println("time interval:", diff.String())
//
//		return nil
//	}
//
//	// 注册任务
//	scheduler.Add(&Clock{})
//	// 运行调度器
//	scheduler.Run()
//	// 检测调度器是否退出
//	<-scheduler.Done()
func NewScheduler(ctx context.Context, lg logger.Iface) *Scheduler {
	s := &Scheduler{
		schedules:   make([]*Schedule, 0),
		msgDisabled: false,
		ctx:         ctx,
		logger:      lg,
	}
	if lg == nil {
		s.logger = logger.NewDefaultLogger()
	}

	return s
}
