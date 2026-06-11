<script>
export default {
  data() {
    return {
      selected: 0,
      list: [
        { pagePath: "/pages/index/index", text: "首页", icon: "" },
        { pagePath: "/pages/growth/index", text: "成长", icon: "" },
        { pagePath: "/pages/tasks/index", text: "任务", icon: "" },
        { pagePath: "/pages/profile/index", text: "我的", icon: "" }
      ]
    }
  },
  created() {
    this.updateSelected()
  },
  methods: {
    updateSelected() {
      const pages = getCurrentPages()
      if (pages.length > 0) {
        const current = "/" + pages[pages.length - 1].route
        const idx = this.list.findIndex(item => item.pagePath === current)
        if (idx !== -1) this.selected = idx
      }
    },
    switchTab(e) {
      const item = e.currentTarget.dataset.item
      const url = "/" + item.pagePath
      uni.switchTab({ url })
    }
  }
}
</script>

<template>
  <view class="tab-bar">
    <view
      v-for="(item, index) in list"
      :key="index"
      class="tab-bar-item"
      :class="{ active: index === selected }"
      :data-item="item"
      @click="switchTab"
    >
      <text class="tab-icon">{{ item.icon }}</text>
      <text class="tab-text">{{ item.text }}</text>
    </view>
  </view>
</template>

<style scoped>
.tab-bar {
  position: fixed;
  bottom: 0;
  left: 0;
  right: 0;
  display: flex;
  justify-content: space-around;
  align-items: center;
  height: 100rpx;
  padding-bottom: env(safe-area-inset-bottom);
  background: #FFFFFF;
  box-shadow: 0 -1px 2px rgba(0, 0, 0, 0.05);
  z-index: 999;
}

.tab-bar-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  flex: 1;
  height: 100%;
}

.tab-icon {
  font-family: remixicon;
  font-size: 48rpx;
  color: #94A3B8;
  line-height: 1.2;
}

.tab-text {
  margin-top: 4rpx;
  font-size: $typography-font-size-xs;
  color: #94A3B8;
  line-height: 1.2;
}

.tab-bar-item.active .tab-icon,
.tab-bar-item.active .tab-text {
  color: #F97316;
  font-weight: 500;
}
</style>
